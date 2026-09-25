package lifecycle

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) ReleaseManaged(ctx context.Context, organizationID, numberID uuid.UUID, key string) (sqlc.NumberLifecycleOperation, error) {
	key = strings.TrimSpace(key)
	if err := validateLifecycleKey(organizationID, key); err != nil {
		return sqlc.NumberLifecycleOperation{}, err
	}
	if existing, err := s.repo.GetLifecycleOperationByKey(ctx, organizationID, key); err == nil {
		if existing.Operation != "release" || existing.PhoneNumberID == nil || *existing.PhoneNumberID != numberID {
			return sqlc.NumberLifecycleOperation{}, apperror.NewConflict("idempotency key was used for another operation")
		}
		return existing, nil
	}
	if s.db == nil || s.lifecycle == nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewServiceUnavailable("managed number lifecycle is not configured", nil)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("begin managed release", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	number, err := queries.GetPhoneNumberForRelease(ctx, sqlc.GetPhoneNumberForReleaseParams{
		ID:             numberID,
		OrganizationID: organizationID,
	})
	if err != nil || number.ProvisioningMode != "managed" || number.ProviderResourceID == nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewNotFound("managed number not found")
	}
	op, err := queries.CreateManagedReleaseOperation(ctx, sqlc.CreateManagedReleaseOperationParams{
		OrganizationID: organizationID,
		PhoneNumberID:   numberID,
		IdempotencyKey:  key,
	})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, writeError(err)
	}
	if _, err = queries.DisableManagedPhoneNumberForRelease(ctx, sqlc.DisableManagedPhoneNumberForReleaseParams{
		ID:             numberID,
		OrganizationID: organizationID,
	}); err != nil {
		return sqlc.NumberLifecycleOperation{}, writeError(err)
	}
	// Record that provider submission may have started before performing the external request.
	// A worker restart must never turn a pending operation into a second termination request.
	op, err = queries.MarkNumberLifecycleSubmitted(ctx, sqlc.MarkNumberLifecycleSubmittedParams{
		ID:             op.ID,
		ReconcileAfter: pgTimestamptz(s.now().Add(managedReconcileDelay)),
	})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("claim managed release submission", err)
	}
	if err = insertNumberEvent(ctx, queries, "number.release.requested", organizationID, op.ID, op); err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("enqueue number release event", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("commit managed release", err)
	}
	reference, providerErr := s.lifecycle.RequestRelease(ctx, *number.ProviderResourceID)
	var referencePointer *string
	if reference != "" {
		referencePointer = &reference
	}
	op, updateErr := s.repo.MarkLifecycleSubmitted(ctx, op.ID, referencePointer, s.now().Add(managedReconcileDelay))
	if updateErr != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("schedule managed release reconciliation", updateErr)
	}
	if providerErr != nil && !errors.Is(providerErr, ErrProviderCapabilityUnavailable) {
		return op, nil
	}
	return op, nil
}

func (s *Service) reconcileRelease(ctx context.Context, operation sqlc.NumberLifecycleOperation) error {
	if operation.PhoneNumberID == nil || s.lifecycle == nil {
		return fmt.Errorf("release operation dependencies are incomplete")
	}
	number, err := s.repo.GetForRelease(ctx, operation.OrganizationID, *operation.PhoneNumberID)
	if errors.Is(err, pgx.ErrNoRows) && operation.Status == "completed" {
		return nil
	}
	if err != nil {
		return err
	}
	if number.ProviderResourceID == nil {
		return fmt.Errorf("managed number provider identity is missing")
	}
	// Pending operations from earlier versions may already have reached DIDWW.
	// Verify provider state; never infer that another termination request is safe.
	completed, providerErr := s.lifecycle.ReleaseCompleted(ctx, *number.ProviderResourceID)
	if providerErr != nil || !completed {
		_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	if _, err = queries.ReleaseManagedPhoneNumber(ctx, *operation.PhoneNumberID); err != nil {
		return err
	}
	completedOperation, err := queries.CompleteNumberLifecycleOperation(ctx, operation.ID)
	if err != nil {
		return err
	}
	if err = insertNumberEvent(ctx, queries, "number.released", operation.OrganizationID, operation.ID, completedOperation); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

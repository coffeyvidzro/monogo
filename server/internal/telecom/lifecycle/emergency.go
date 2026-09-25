package lifecycle

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) PutEmergency(ctx context.Context, organizationID, numberID uuid.UUID, key string, req EmergencyAddressRequest) (sqlc.EmergencyRegistration, error) {
	key = strings.TrimSpace(key)
	if err := validateLifecycleKey(organizationID, key); err != nil {
		return sqlc.EmergencyRegistration{}, err
	}
	normalizeEmergency(&req)
	if err := validateEmergency(req); err != nil {
		return sqlc.EmergencyRegistration{}, err
	}
	payload, _ := json.Marshal(req)
	digest := sha256.Sum256(payload)
	hash := hex.EncodeToString(digest[:])
	if existing, err := s.repo.GetEmergencyByKey(ctx, organizationID, key); err == nil {
		if existing.RequestHash == nil || *existing.RequestHash != hash {
			return sqlc.EmergencyRegistration{}, apperror.NewConflict("idempotency key was used with another emergency address")
		}
		return existing, nil
	}
	if s.db == nil {
		return sqlc.EmergencyRegistration{}, apperror.NewServiceUnavailable("emergency registration is not configured", nil)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return sqlc.EmergencyRegistration{}, apperror.NewInternal("begin emergency registration", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	_ = queries.DeactivateCurrentEmergencyRegistration(ctx, sqlc.DeactivateCurrentEmergencyRegistrationParams{OrganizationID: organizationID, PhoneNumberID: numberID})
	line2 := req.AddressLine2
	registration, err := queries.CreateEmergencyRegistration(ctx, sqlc.CreateEmergencyRegistrationParams{
		OrganizationID: organizationID, PhoneNumberID: numberID, IdempotencyKey: &key, RequestHash: &hash,
		Name: req.Name, AddressLine1: req.AddressLine1, AddressLine2: &line2,
		Locality: req.Locality, Region: req.Region, PostalCode: req.PostalCode, CountryCode: req.CountryCode,
	})
	if err != nil {
		return sqlc.EmergencyRegistration{}, writeError(err)
	}
	if err = insertNumberEvent(ctx, queries, "number.e911.requested", organizationID, registration.ID, registration); err != nil {
		return sqlc.EmergencyRegistration{}, apperror.NewInternal("enqueue emergency registration event", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return sqlc.EmergencyRegistration{}, apperror.NewInternal("commit emergency registration", err)
	}
	return registration, nil
}

func (s *Service) GetEmergency(ctx context.Context, organizationID, numberID uuid.UUID) (sqlc.EmergencyRegistration, error) {
	registration, err := s.repo.GetCurrentEmergency(ctx, organizationID, numberID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.EmergencyRegistration{}, apperror.NewNotFound("emergency registration not found")
	}
	if err != nil {
		return sqlc.EmergencyRegistration{}, apperror.NewInternal("get emergency registration", err)
	}
	return registration, nil
}

func (s *Service) ReconcileEmergency(ctx context.Context, registration sqlc.EmergencyRegistration) error {
	if s.lifecycle == nil {
		return fmt.Errorf("emergency provider is not configured")
	}
	number, err := s.repo.Get(ctx, registration.OrganizationID, registration.PhoneNumberID)
	if err != nil {
		return err
	}
	address := EmergencyAddressRequest{
		Name: registration.Name, AddressLine1: registration.AddressLine1,
		Locality: registration.Locality, Region: registration.Region,
		PostalCode: registration.PostalCode, CountryCode: registration.CountryCode,
	}
	if registration.AddressLine2 != nil {
		address.AddressLine2 = *registration.AddressLine2
	}
	reference, valid, message, providerErr := s.lifecycle.ValidateEmergency(ctx, number.Number, address)
	if providerErr != nil {
		_, err = s.repo.queries.MarkEmergencyRegistrationValidating(ctx, sqlc.MarkEmergencyRegistrationValidatingParams{
			ID: registration.ID, ValidationMessage: stringPointer(providerErr.Error()),
			ReconcileAfter: pgTimestamptz(s.now().Add(managedReconcileDelay)),
		})
		return err
	}
	if !valid {
		rejected, rejectErr := s.repo.queries.RejectEmergencyRegistration(ctx, sqlc.RejectEmergencyRegistrationParams{ID: registration.ID, ValidationMessage: &message})
		if rejectErr != nil {
			return rejectErr
		}
		return insertNumberEvent(ctx, s.repo.queries, "number.e911.rejected", registration.OrganizationID, registration.ID, rejected)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	activated, err := queries.ActivateEmergencyRegistration(ctx, sqlc.ActivateEmergencyRegistrationParams{ID: registration.ID, ProviderReference: &reference})
	if err != nil {
		return err
	}
	if err = insertNumberEvent(ctx, queries, "number.e911.activated", registration.OrganizationID, registration.ID, activated); err != nil {
		return err
	}
	return tx.Commit(ctx)
}


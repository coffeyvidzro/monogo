package numbers

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) ReconcileLifecycle(ctx context.Context, operation sqlc.NumberLifecycleOperation) error {
	switch operation.Operation {
	case "release":
		return s.reconcileRelease(ctx, operation)
	case "port_in":
		return s.reconcilePortIn(ctx, operation)
	default:
		return nil
	}
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
	if operation.Status == "pending" {
		reference, providerErr := s.lifecycle.RequestRelease(ctx, *number.ProviderResourceID)
		var pointer *string
		if reference != "" {
			pointer = &reference
		}
		_, err = s.repo.MarkLifecycleSubmitted(ctx, operation.ID, pointer, s.now().Add(managedReconcileDelay))
		if err != nil {
			return err
		}
		if providerErr != nil {
			return nil
		}
	}
	completed, providerErr := s.lifecycle.ReleaseCompleted(ctx, *number.ProviderResourceID)
	if providerErr != nil || !completed {
		_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
		return err
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
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

func (s *Service) reconcilePortIn(ctx context.Context, operation sqlc.NumberLifecycleOperation) error {
	if s.lifecycle == nil {
		return fmt.Errorf("port-in provider is not configured")
	}
	portCase, err := s.repo.queries.GetPortInCaseByOperation(ctx, operation.ID)
	if err != nil {
		return err
	}
	documents, err := s.repo.queries.ListPortInDocuments(ctx, sqlc.ListPortInDocumentsParams{OrganizationID: operation.OrganizationID, PortInCaseID: portCase.ID})
	if err != nil {
		return err
	}
	var request PortInRequest
	if err := json.Unmarshal(operation.RequestPayload, &request); err != nil {
		return err
	}
	if portCase.Status == "draft" || portCase.Status == "checking_portability" {
		portable, reason, portabilityErr := s.lifecycle.CheckPortability(ctx, request.Number)
		if portabilityErr != nil {
			_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
			return err
		}
		if !portable {
			if strings.TrimSpace(reason) == "" {
				reason = "number is not portable"
			}
			code := "not_portable"
			if _, err = s.repo.queries.RejectPortInCase(ctx, sqlc.RejectPortInCaseParams{ID: portCase.ID, RejectionCode: &code, RejectionMessage: &reason}); err != nil {
				return err
			}
			if _, err = s.repo.queries.FailNumberLifecycleOperation(ctx, sqlc.FailNumberLifecycleOperationParams{ID: operation.ID, FailureCode: &code, FailureMessage: &reason}); err != nil {
				return err
			}
			return nil
		}
		portCase, err = s.repo.queries.MarkPortInDocumentsRequired(ctx, portCase.ID)
		if err != nil {
			return err
		}
	}
	if operation.ProviderReference == nil {
		if len(documents) == 0 {
			_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
			return err
		}
		reference, providerErr := s.lifecycle.SubmitPortIn(ctx, operation.ID, request, documents)
		if providerErr != nil {
			_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
			return err
		}
		if _, err = s.repo.queries.MarkPortInSubmitted(ctx, sqlc.MarkPortInSubmittedParams{ID: portCase.ID, ProviderCaseReference: &reference}); err != nil {
			return err
		}
		_, err = s.repo.MarkLifecycleSubmitted(ctx, operation.ID, &reference, s.now().Add(managedReconcileDelay))
		return err
	}
	status, providerErr := s.lifecycle.PortInStatus(ctx, *operation.ProviderReference)
	if providerErr != nil {
		_, err = s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
		return err
	}
	if status.Status == "rejected" {
		message := strings.TrimSpace(status.RejectionMessage)
		if message == "" {
			message = "provider rejected the port-in request"
		}
		code := strings.TrimSpace(status.RejectionCode)
		if code == "" {
			code = "provider_rejected"
		}
		rejected, rejectErr := s.repo.queries.RejectPortInCase(ctx, sqlc.RejectPortInCaseParams{ID: portCase.ID, RejectionCode: &code, RejectionMessage: &message})
		if rejectErr != nil {
			return rejectErr
		}
		if _, failErr := s.repo.queries.FailNumberLifecycleOperation(ctx, sqlc.FailNumberLifecycleOperationParams{ID: operation.ID, FailureCode: &code, FailureMessage: &message}); failErr != nil {
			return failErr
		}
		return insertNumberEvent(ctx, s.repo.queries, "number.port_in.rejected", operation.OrganizationID, portCase.ID, rejected)
	}
	if status.Status == "foc_received" && status.FOCAt != nil {
		_, err = s.repo.queries.MarkPortInFOC(ctx, sqlc.MarkPortInFOCParams{ID: portCase.ID, FocAt: pgTimestamptz(*status.FOCAt)})
	}
	if status.Status != "activated" {
		_, scheduleErr := s.repo.ScheduleLifecycle(ctx, operation.ID, s.now().Add(managedReconcileDelay))
		return errors.Join(err, scheduleErr)
	}
	if status.ProviderResourceID == "" {
		return fmt.Errorf("activated port is missing provider resource identity")
	}
	return s.activatePortIn(ctx, operation, portCase, request, status.ProviderResourceID)
}

func (s *Service) activatePortIn(ctx context.Context, operation sqlc.NumberLifecycleOperation, portCase sqlc.PortInCase, request PortInRequest, providerResourceID string) error {
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	number, err := queries.CreateManagedPhoneNumber(ctx, sqlc.CreateManagedPhoneNumberParams{
		OrganizationID: operation.OrganizationID, Number: request.Number,
		CountryCode: request.ServiceAddress.CountryCode, ProviderID: &operation.ProviderID,
		ProviderResourceID: &providerResourceID,
	})
	if err != nil {
		return err
	}
	if _, err = queries.ActivatePortInCase(ctx, portCase.ID); err != nil {
		return err
	}
	completed, err := queries.CompleteNumberLifecycleOperation(ctx, operation.ID)
	if err != nil {
		return err
	}
	if err = insertNumberEvent(ctx, queries, "number.port_in.activated", operation.OrganizationID, portCase.ID, map[string]any{"operation": completed, "number": number}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func stringPointer(value string) *string { return &value }
func pgTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: value, Valid: true}
}

package lifecycle

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

func (s *Service) CreatePortIn(ctx context.Context, organizationID uuid.UUID, key string, req PortInRequest) (PortInResponse, error) {
	key = strings.TrimSpace(key)
	req.Number = strings.TrimSpace(req.Number)
	req.LosingCarrier = strings.TrimSpace(req.LosingCarrier)
	req.AccountNumber = strings.TrimSpace(req.AccountNumber)
	req.AuthorizedName = strings.TrimSpace(req.AuthorizedName)
	normalizeEmergency(&req.ServiceAddress)
	if err := validateLifecycleKey(organizationID, key); err != nil {
		return PortInResponse{}, err
	}
	if !e164.MatchString(req.Number) || req.LosingCarrier == "" || req.AccountNumber == "" || req.AuthorizedName == "" {
		return PortInResponse{}, apperror.NewBadRequest("number and porting account details are required")
	}
	if err := validateEmergency(req.ServiceAddress); err != nil {
		return PortInResponse{}, err
	}
	if existing, err := s.repo.GetLifecycleOperationByKey(ctx, organizationID, key); err == nil {
		if existing.Operation != "port_in" || existing.RequestedNumber != req.Number {
			return PortInResponse{}, apperror.NewConflict("idempotency key was used for another operation")
		}
		portCase, err := s.repo.queries.GetPortInCaseByOperation(ctx, existing.ID)
		if err != nil {
			return PortInResponse{}, apperror.NewInternal("get port-in case", err)
		}
		documents, err := s.repo.queries.ListPortInDocuments(ctx, sqlc.ListPortInDocumentsParams{
			OrganizationID: organizationID,
			PortInCaseID:   portCase.ID,
		})
		return PortInResponse{
			Case:      portCase,
			Operation: existing,
			Documents: documents,
		}, err
	}
	if s.db == nil {
		return PortInResponse{}, apperror.NewServiceUnavailable("port-in lifecycle is not configured", nil)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
	if err != nil {
		return PortInResponse{}, apperror.NewInternal("begin port-in", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repository := s.repo.WithQueries(sqlc.New(tx))
	op, portCase, err := repository.CreatePortIn(ctx, organizationID, key, req)
	if err != nil {
		return PortInResponse{}, writeError(err)
	}
	if err = insertNumberEvent(ctx, sqlc.New(tx), "number.port_in.created", organizationID, portCase.ID, portCase); err != nil {
		return PortInResponse{}, apperror.NewInternal("enqueue port-in event", err)
	}
	if err = tx.Commit(ctx); err != nil {
		return PortInResponse{}, apperror.NewInternal("commit port-in", err)
	}
	return PortInResponse{
		Case:      portCase,
		Operation: op,
		Documents: []sqlc.PortInDocument{},
	}, nil
}

func (s *Service) GetPortIn(ctx context.Context, organizationID, caseID uuid.UUID) (PortInResponse, error) {
	portCase, documents, err := s.repo.GetPortIn(ctx, organizationID, caseID)
	if errors.Is(err, pgx.ErrNoRows) {
		return PortInResponse{}, apperror.NewNotFound("port-in case not found")
	}
	if err != nil {
		return PortInResponse{}, apperror.NewInternal("get port-in case", err)
	}
	op, err := s.repo.GetLifecycleOperation(ctx, organizationID, portCase.LifecycleOperationID)
	if err != nil {
		return PortInResponse{}, apperror.NewInternal("get port-in operation", err)
	}
	return PortInResponse{
		Case:      portCase,
		Operation: op,
		Documents: documents,
	}, nil
}

func (s *Service) AddPortDocument(ctx context.Context, organizationID, caseID uuid.UUID, req PortDocumentRequest) (sqlc.PortInDocument, error) {
	req.DocumentType = strings.ToLower(strings.TrimSpace(req.DocumentType))
	req.ObjectKey = strings.TrimSpace(req.ObjectKey)
	req.SHA256 = strings.ToLower(strings.TrimSpace(req.SHA256))
	if organizationID == uuid.Nil || caseID == uuid.Nil {
		return sqlc.PortInDocument{}, apperror.NewBadRequest("organization and port-in case are required")
	}
	if req.ObjectKey == "" || !regexp.MustCompile(`^[0-9a-f]{64}$`).MatchString(req.SHA256) {
		return sqlc.PortInDocument{}, apperror.NewBadRequest("document object_key and sha256 are required")
	}
	switch req.DocumentType {
	case "loa", "invoice", "ownership", "identity", "other":
	default:
		return sqlc.PortInDocument{}, apperror.NewBadRequest("document_type is invalid")
	}
	document, err := s.repo.AddPortDocument(ctx, organizationID, caseID, req)
	if err != nil {
		return sqlc.PortInDocument{}, writeError(err)
	}
	return document, nil
}

func (s *Service) reconcilePortIn(ctx context.Context, operation sqlc.NumberLifecycleOperation) error {
	if s.lifecycle == nil {
		return fmt.Errorf("port-in provider is not configured")
	}
	portCase, err := s.repo.queries.GetPortInCaseByOperation(ctx, operation.ID)
	if err != nil {
		return err
	}
	documents, err := s.repo.queries.ListPortInDocuments(ctx, sqlc.ListPortInDocumentsParams{
		OrganizationID: operation.OrganizationID,
		PortInCaseID:   portCase.ID,
	})
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
			if _, err = s.repo.queries.RejectPortInCase(ctx, sqlc.RejectPortInCaseParams{
				ID:               portCase.ID,
				RejectionCode:    &code,
				RejectionMessage: &reason,
			}); err != nil {
				return err
			}
			if _, err = s.repo.queries.FailNumberLifecycleOperation(ctx, sqlc.FailNumberLifecycleOperationParams{
				ID:             operation.ID,
				FailureCode:    &code,
				FailureMessage: &reason,
			}); err != nil {
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
		if _, err = s.repo.queries.MarkPortInSubmitted(ctx, sqlc.MarkPortInSubmittedParams{
			ID:                    portCase.ID,
			ProviderCaseReference: &reference,
		}); err != nil {
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
		rejected, rejectErr := s.repo.queries.RejectPortInCase(ctx, sqlc.RejectPortInCaseParams{
			ID:               portCase.ID,
			RejectionCode:    &code,
			RejectionMessage: &message,
		})
		if rejectErr != nil {
			return rejectErr
		}
		if _, failErr := s.repo.queries.FailNumberLifecycleOperation(ctx, sqlc.FailNumberLifecycleOperationParams{
			ID:             operation.ID,
			FailureCode:    &code,
			FailureMessage: &message,
		}); failErr != nil {
			return failErr
		}
		return insertNumberEvent(ctx, s.repo.queries, "number.port_in.rejected", operation.OrganizationID, portCase.ID, rejected)
	}
	if status.Status == "foc_received" && status.FOCAt != nil {
		_, err = s.repo.queries.MarkPortInFOC(ctx, sqlc.MarkPortInFOCParams{
			ID:    portCase.ID,
			FocAt: pgTimestamptz(*status.FOCAt),
		})
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
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{
		IsoLevel: pgx.Serializable,
	})
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
	if err = insertNumberEvent(ctx, queries, "number.port_in.activated", operation.OrganizationID, portCase.ID, map[string]any{
		"operation": completed,
		"number":    number,
	}); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func stringPointer(value string) *string {
	return &value
}

func pgTimestamptz(value time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{
		Time:  value,
		Valid: true,
	}
}

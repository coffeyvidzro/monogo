package numbers

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"regexp"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var postalCode = regexp.MustCompile(`^[A-Z0-9][A-Z0-9 -]{1,11}$`)

func (s *Service) ConfigureLifecycle(provider NumberLifecycleProvider) { s.lifecycle = provider }

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
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("begin managed release", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := sqlc.New(tx)
	number, err := queries.GetPhoneNumberForRelease(ctx, sqlc.GetPhoneNumberForReleaseParams{ID: numberID, OrganizationID: organizationID})
	if err != nil || number.ProvisioningMode != "managed" || number.ProviderResourceID == nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewNotFound("managed number not found")
	}
	op, err := queries.CreateManagedReleaseOperation(ctx, sqlc.CreateManagedReleaseOperationParams{OrganizationID: organizationID, PhoneNumberID: numberID, IdempotencyKey: key})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, writeError(err)
	}
	if _, err = queries.DisableManagedPhoneNumberForRelease(ctx, sqlc.DisableManagedPhoneNumberForReleaseParams{ID: numberID, OrganizationID: organizationID}); err != nil {
		return sqlc.NumberLifecycleOperation{}, writeError(err)
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

func (s *Service) GetLifecycle(ctx context.Context, organizationID, operationID uuid.UUID) (sqlc.NumberLifecycleOperation, error) {
	op, err := s.repo.GetLifecycleOperation(ctx, organizationID, operationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.NumberLifecycleOperation{}, apperror.NewNotFound("number lifecycle operation not found")
	}
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, apperror.NewInternal("get number lifecycle operation", err)
	}
	return op, nil
}

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
		documents, err := s.repo.queries.ListPortInDocuments(ctx, sqlc.ListPortInDocumentsParams{OrganizationID: organizationID, PortInCaseID: portCase.ID})
		return PortInResponse{Case: portCase, Operation: existing, Documents: documents}, err
	}
	if s.db == nil {
		return PortInResponse{}, apperror.NewServiceUnavailable("port-in lifecycle is not configured", nil)
	}
	tx, err := s.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
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
	return PortInResponse{Case: portCase, Operation: op, Documents: []sqlc.PortInDocument{}}, nil
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
	return PortInResponse{Case: portCase, Operation: op, Documents: documents}, nil
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

func validateLifecycleKey(organizationID uuid.UUID, key string) error {
	if organizationID == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	if len(key) < 1 || len(key) > 255 {
		return apperror.NewBadRequest("Idempotency-Key is required and must not exceed 255 characters")
	}
	return nil
}

func normalizeEmergency(req *EmergencyAddressRequest) {
	req.Name = strings.TrimSpace(req.Name)
	req.AddressLine1 = strings.TrimSpace(req.AddressLine1)
	req.AddressLine2 = strings.TrimSpace(req.AddressLine2)
	req.Locality = strings.TrimSpace(req.Locality)
	req.Region = strings.ToUpper(strings.TrimSpace(req.Region))
	req.PostalCode = strings.ToUpper(strings.TrimSpace(req.PostalCode))
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
}

func validateEmergency(req EmergencyAddressRequest) error {
	if req.Name == "" || req.AddressLine1 == "" || req.Locality == "" || req.Region == "" {
		return apperror.NewBadRequest("complete emergency address is required")
	}
	if len(req.CountryCode) != 2 {
		return apperror.NewBadRequest("country_code must be a two-letter ISO country code")
	}
	if !postalCode.MatchString(req.PostalCode) {
		return apperror.NewBadRequest("postal_code is invalid")
	}
	return nil
}

func insertNumberEvent(ctx context.Context, queries *sqlc.Queries, subject string, organizationID, aggregateID uuid.UUID, resource any) error {
	payload := map[string]any{"event_type": subject, "organization_id": organizationID, "resource": resource, "occurred_at": time.Now().UTC()}
	_, err := outbox.NewRepository(queries).Insert(ctx, outbox.Event{
		Subject: subject, AggregateType: "number_lifecycle", AggregateID: aggregateID,
		Payload: payload, Headers: map[string]string{"event_type": subject, "organization_id": organizationID.String()},
	})
	return err
}

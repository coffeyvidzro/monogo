package numbers

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

func (r *Repository) GetForRelease(ctx context.Context, organizationID, numberID uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberForRelease(ctx, sqlc.GetPhoneNumberForReleaseParams{
		ID: numberID, OrganizationID: organizationID,
	})
}

func (r *Repository) CreateReleaseOperation(ctx context.Context, organizationID, numberID uuid.UUID, key string) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.CreateManagedReleaseOperation(ctx, sqlc.CreateManagedReleaseOperationParams{
		OrganizationID: organizationID, PhoneNumberID: numberID, IdempotencyKey: key,
	})
}

func (r *Repository) GetLifecycleOperation(ctx context.Context, organizationID, id uuid.UUID) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.GetNumberLifecycleOperation(ctx, sqlc.GetNumberLifecycleOperationParams{ID: id, OrganizationID: organizationID})
}

func (r *Repository) GetLifecycleOperationByKey(ctx context.Context, organizationID uuid.UUID, key string) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.GetNumberLifecycleOperationByKey(ctx, sqlc.GetNumberLifecycleOperationByKeyParams{OrganizationID: organizationID, IdempotencyKey: key})
}

func (r *Repository) ListLifecycleDue(ctx context.Context, batch int32) ([]sqlc.NumberLifecycleOperation, error) {
	return r.queries.ListNumberLifecycleOperationsDue(ctx, batch)
}

func (r *Repository) MarkLifecycleSubmitted(ctx context.Context, id uuid.UUID, reference *string, next time.Time) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.MarkNumberLifecycleSubmitted(ctx, sqlc.MarkNumberLifecycleSubmittedParams{
		ID: id, ProviderReference: reference, ReconcileAfter: pgconv.TimeToTimestamptz(next),
	})
}

func (r *Repository) ScheduleLifecycle(ctx context.Context, id uuid.UUID, next time.Time) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.ScheduleNumberLifecycleReconciliation(ctx, sqlc.ScheduleNumberLifecycleReconciliationParams{
		ID: id, ReconcileAfter: pgconv.TimeToTimestamptz(next),
	})
}

func (r *Repository) CompleteLifecycle(ctx context.Context, id uuid.UUID) (sqlc.NumberLifecycleOperation, error) {
	return r.queries.CompleteNumberLifecycleOperation(ctx, id)
}

func (r *Repository) CreateEmergency(ctx context.Context, organizationID, numberID uuid.UUID, key, hash string, req EmergencyAddressRequest) (sqlc.EmergencyRegistration, error) {
	addressLine2 := req.AddressLine2
	return r.queries.CreateEmergencyRegistration(ctx, sqlc.CreateEmergencyRegistrationParams{
		OrganizationID: organizationID, PhoneNumberID: numberID,
		IdempotencyKey: &key, RequestHash: &hash, Name: req.Name,
		AddressLine1: req.AddressLine1, AddressLine2: &addressLine2,
		Locality: req.Locality, Region: req.Region, PostalCode: req.PostalCode,
		CountryCode: req.CountryCode,
	})
}

func (r *Repository) GetEmergencyByKey(ctx context.Context, organizationID uuid.UUID, key string) (sqlc.EmergencyRegistration, error) {
	return r.queries.GetEmergencyRegistrationByKey(ctx, sqlc.GetEmergencyRegistrationByKeyParams{OrganizationID: organizationID, IdempotencyKey: &key})
}

func (r *Repository) GetCurrentEmergency(ctx context.Context, organizationID, numberID uuid.UUID) (sqlc.EmergencyRegistration, error) {
	return r.queries.GetCurrentEmergencyRegistration(ctx, sqlc.GetCurrentEmergencyRegistrationParams{OrganizationID: organizationID, PhoneNumberID: numberID})
}

func (r *Repository) DeactivateEmergency(ctx context.Context, organizationID, numberID uuid.UUID) error {
	return r.queries.DeactivateCurrentEmergencyRegistration(ctx, sqlc.DeactivateCurrentEmergencyRegistrationParams{OrganizationID: organizationID, PhoneNumberID: numberID})
}

func (r *Repository) ListEmergencyDue(ctx context.Context, batch int32) ([]sqlc.EmergencyRegistration, error) {
	return r.queries.ListEmergencyRegistrationsDue(ctx, batch)
}

func (r *Repository) CreatePortIn(ctx context.Context, organizationID uuid.UUID, key string, req PortInRequest) (sqlc.NumberLifecycleOperation, sqlc.PortInCase, error) {
	payload, err := json.Marshal(req)
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, sqlc.PortInCase{}, err
	}
	op, err := r.queries.CreatePortInOperation(ctx, sqlc.CreatePortInOperationParams{
		OrganizationID: organizationID, IdempotencyKey: key, RequestedNumber: req.Number, RequestPayload: payload,
	})
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, sqlc.PortInCase{}, err
	}
	address, err := json.Marshal(req.ServiceAddress)
	if err != nil {
		return sqlc.NumberLifecycleOperation{}, sqlc.PortInCase{}, err
	}
	var desired pgtype.Date
	if req.DesiredPortDate != nil {
		desired = pgtype.Date{Time: *req.DesiredPortDate, Valid: true}
	}
	portCase, err := r.queries.CreatePortInCase(ctx, sqlc.CreatePortInCaseParams{
		OrganizationID: organizationID, LifecycleOperationID: op.ID,
		LosingCarrier: req.LosingCarrier, AccountNumber: req.AccountNumber,
		AuthorizedName: req.AuthorizedName, ServiceAddress: address, DesiredPortDate: desired,
	})
	return op, portCase, err
}

func (r *Repository) GetPortIn(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PortInCase, []sqlc.PortInDocument, error) {
	portCase, err := r.queries.GetPortInCase(ctx, sqlc.GetPortInCaseParams{ID: id, OrganizationID: organizationID})
	if err != nil {
		return sqlc.PortInCase{}, nil, err
	}
	documents, err := r.queries.ListPortInDocuments(ctx, sqlc.ListPortInDocumentsParams{OrganizationID: organizationID, PortInCaseID: id})
	return portCase, documents, err
}

func (r *Repository) AddPortDocument(ctx context.Context, organizationID, caseID uuid.UUID, req PortDocumentRequest) (sqlc.PortInDocument, error) {
	return r.queries.AddPortInDocument(ctx, sqlc.AddPortInDocumentParams{
		OrganizationID: organizationID, PortInCaseID: caseID, DocumentType: req.DocumentType,
		ObjectKey: req.ObjectKey, Sha256: req.SHA256,
	})
}

package numbers

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) CreateBYOC(ctx context.Context, organizationID uuid.UUID, req CreateBYOCRequest) (sqlc.PhoneNumber, error) {
	return r.queries.CreateBYOCPhoneNumber(ctx, sqlc.CreateBYOCPhoneNumberParams{
		OrganizationID:      organizationID,
		Number:              req.Number,
		CountryCode:         req.CountryCode,
		CarrierConnectionID: req.CarrierConnectionID,
		VoiceEnabled:        req.VoiceEnabled,
		SmsEnabled:          req.SmsEnabled,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.PhoneNumber, error) {
	return r.queries.ListPhoneNumbersByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberByID(ctx, sqlc.GetPhoneNumberByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (sqlc.PhoneNumber, error) {
	return r.queries.UpdatePhoneNumber(ctx, sqlc.UpdatePhoneNumberParams{
		ID:             id,
		OrganizationID: organizationID,
		VoiceEnabled:   req.VoiceEnabled,
		SmsEnabled:     req.SmsEnabled,
	})
}

func (r *Repository) SetBYOCConnection(ctx context.Context, organizationID, id, connectionID uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.SetBYOCPhoneNumberCarrierConnection(ctx, sqlc.SetBYOCPhoneNumberCarrierConnectionParams{
		ID:                  id,
		OrganizationID:      organizationID,
		CarrierConnectionID: &connectionID,
	})
}

func (r *Repository) ReleaseBYOC(ctx context.Context, organizationID, id uuid.UUID) (sqlc.PhoneNumber, error) {
	return r.queries.ReleaseBYOCPhoneNumber(ctx, sqlc.ReleaseBYOCPhoneNumberParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) WithQueries(queries *sqlc.Queries) *Repository {
	return NewRepository(queries)
}

func (r *Repository) ManagedRoutingTargets(
	ctx context.Context,
) (sqlc.CarrierProvider, []sqlc.ListProviderRoutingTargetsRow, error) {
	provider, err := r.queries.GetCarrierProviderBySlug(ctx, "didww")
	if err != nil {
		return sqlc.CarrierProvider{}, nil, err
	}
	targets, err := r.ManagedRoutingTargetsForProvider(ctx, provider.ID)
	return provider, targets, err
}

func (r *Repository) ManagedRoutingTargetsForProvider(
	ctx context.Context,
	providerID uuid.UUID,
) ([]sqlc.ListProviderRoutingTargetsRow, error) {
	return r.queries.ListProviderRoutingTargets(ctx, providerID)
}

func (r *Repository) CreateManagedOrder(ctx context.Context, organizationID uuid.UUID, key, hash string, request ManagedPurchaseRequest, availableDIDID, skuID string) (ManagedOrder, bool, error) {
	row, err := r.queries.CreateManagedNumberOrder(ctx, sqlc.CreateManagedNumberOrderParams{
		ID: uuid.New(), OrganizationID: organizationID, IdempotencyKey: key,
		RequestHash: hash, Number: request.Number, CountryCode: request.CountryCode,
		AvailableDidID: availableDIDID, SkuID: skuID,
	})
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = r.queries.GetManagedNumberOrderByKey(ctx, sqlc.GetManagedNumberOrderByKeyParams{
			OrganizationID: organizationID, IdempotencyKey: key,
		})
	}
	return managedOrder(row), created, err
}

func (r *Repository) GetManagedOrder(ctx context.Context, organizationID, id uuid.UUID) (ManagedOrder, error) {
	row, err := r.queries.GetManagedNumberOrder(ctx, sqlc.GetManagedNumberOrderParams{OrganizationID: organizationID, ID: id})
	return managedOrder(row), err
}

func (r *Repository) GetManagedOrderByKey(
	ctx context.Context,
	organizationID uuid.UUID,
	idempotencyKey string,
) (ManagedOrder, error) {
	row, err := r.queries.GetManagedNumberOrderByKey(
		ctx,
		sqlc.GetManagedNumberOrderByKeyParams{
			OrganizationID: organizationID,
			IdempotencyKey: idempotencyKey,
		},
	)
	return managedOrder(row), err
}

func (r *Repository) GetManagedOrderInternal(ctx context.Context, id uuid.UUID) (ManagedOrder, error) {
	row, err := r.queries.GetManagedNumberOrderInternal(ctx, id)
	return managedOrder(row), err
}

func (r *Repository) ListManagedOrdersDue(ctx context.Context, limit int32) ([]ManagedOrder, error) {
	rows, err := r.queries.ListManagedNumberOrdersForReconciliation(ctx, limit)
	if err != nil {
		return nil, err
	}
	orders := make([]ManagedOrder, 0, len(rows))
	for _, row := range rows {
		orders = append(orders, managedOrder(row))
	}
	return orders, nil
}

func (r *Repository) ClaimManagedSubmission(ctx context.Context, id uuid.UUID) (ManagedOrder, error) {
	row, err := r.queries.ClaimManagedNumberOrderSubmission(ctx, id)
	return managedOrder(row), err
}

func (r *Repository) MarkManagedOutcomeUnknown(ctx context.Context, id uuid.UUID, code, message string, next time.Time) (ManagedOrder, error) {
	row, err := r.queries.MarkManagedNumberOrderOutcomeUnknown(ctx, sqlc.MarkManagedNumberOrderOutcomeUnknownParams{
		ID: id, ErrorCode: &code, ErrorMessage: &message, ReconcileAfter: pgconv.TimeToTimestamptz(next),
	})
	return managedOrder(row), err
}

func (r *Repository) RecordManagedProviderOrder(ctx context.Context, id uuid.UUID, providerOrderID string, next time.Time) (ManagedOrder, error) {
	row, err := r.queries.RecordManagedNumberProviderOrder(ctx, sqlc.RecordManagedNumberProviderOrderParams{
		ID: id, ProviderOrderID: &providerOrderID, ReconcileAfter: pgconv.TimeToTimestamptz(next),
	})
	return managedOrder(row), err
}

func (r *Repository) RecordManagedOwnedDID(ctx context.Context, id uuid.UUID, didID string, verifiedAt time.Time) (ManagedOrder, error) {
	row, err := r.queries.RecordManagedNumberOwnedDID(ctx, sqlc.RecordManagedNumberOwnedDIDParams{
		ID: id, ProviderDidID: &didID, VerifiedAt: pgconv.TimeToTimestamptz(verifiedAt),
	})
	return managedOrder(row), err
}

func (r *Repository) ScheduleManagedReconciliation(ctx context.Context, id uuid.UUID, code string, next time.Time) (ManagedOrder, error) {
	row, err := r.queries.ScheduleManagedNumberReconciliation(ctx, sqlc.ScheduleManagedNumberReconciliationParams{
		ID: id, ErrorCode: &code, ReconcileAfter: pgconv.TimeToTimestamptz(next),
	})
	return managedOrder(row), err
}

func (r *Repository) MarkManagedManualReview(ctx context.Context, id uuid.UUID, code string) (ManagedOrder, error) {
	row, err := r.queries.MarkManagedNumberOrderManualReview(ctx, sqlc.MarkManagedNumberOrderManualReviewParams{ID: id, ErrorCode: &code})
	return managedOrder(row), err
}

func (r *Repository) LockManagedOrder(ctx context.Context, id uuid.UUID) (ManagedOrder, error) {
	row, err := r.queries.LockManagedNumberOrder(ctx, id)
	return managedOrder(row), err
}

func (r *Repository) CreateActivatedManagedNumber(
	ctx context.Context,
	order ManagedOrder,
	didID string,
	carrierConnectionID uuid.UUID,
) (sqlc.PhoneNumber, error) {
	return r.queries.CreateManagedPhoneNumber(ctx, sqlc.CreateManagedPhoneNumberParams{
		OrganizationID: order.OrganizationID, Number: order.Number, CountryCode: order.CountryCode,
		ProviderID: &order.ProviderID, ProviderResourceID: &didID,
		CarrierConnectionID: &carrierConnectionID,
	})
}

func (r *Repository) CompleteManagedOrder(ctx context.Context, id, phoneNumberID uuid.UUID, trunkID string, verifiedAt time.Time) (ManagedOrder, error) {
	row, err := r.queries.CompleteManagedNumberOrder(ctx, sqlc.CompleteManagedNumberOrderParams{
		ID: id, PhoneNumberID: &phoneNumberID, InboundTrunkID: &trunkID,
		VerifiedAt: pgconv.TimeToTimestamptz(verifiedAt),
	})
	return managedOrder(row), err
}

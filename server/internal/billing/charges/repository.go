package charges

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	if queries == nil {
		panic("billing charges: queries are required")
	}
	return &Repository{queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Charge, error) {
	return r.queries.CreateBillingCharge(ctx, sqlc.CreateBillingChargeParams{
		OrganizationID:  req.OrganizationID,
		WalletID:        req.WalletID,
		ResourceType:    req.ResourceType,
		ResourceID:      req.ResourceID,
		ChargingMode:    req.ChargingMode,
		Currency:        req.Currency,
		IdempotencyKey:  req.IdempotencyKey,
		RequestHash:     req.RequestHash,
		PricingSnapshot: req.PricingSnapshot,
	})
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Charge, error) {
	return r.queries.GetBillingCharge(ctx, sqlc.GetBillingChargeParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) ByIdempotencyKey(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
) (sqlc.Charge, error) {
	return r.queries.GetBillingChargeByIdempotencyKey(
		ctx,
		sqlc.GetBillingChargeByIdempotencyKeyParams{
			OrganizationID: organizationID,
			IdempotencyKey: key,
		},
	)
}

func (r *Repository) ByResource(
	ctx context.Context,
	organizationID uuid.UUID,
	resourceType string,
	resourceID uuid.UUID,
) (sqlc.Charge, error) {
	return r.queries.GetBillingChargeByResource(ctx, sqlc.GetBillingChargeByResourceParams{
		OrganizationID: organizationID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
	})
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	req ListRequest,
) ([]sqlc.Charge, error) {
	return r.queries.ListBillingCharges(ctx, sqlc.ListBillingChargesParams{
		OrganizationID: organizationID,
		Status:         req.Status,
		PageLimit:      req.Limit,
		PageOffset:     req.Offset,
	})
}

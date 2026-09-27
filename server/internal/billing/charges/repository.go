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

	return &Repository{
		queries: queries,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Charge, error) {
	params := sqlc.CreateBillingChargeParams{
		OrganizationID:  req.OrganizationID,
		WalletID:        req.WalletID,
		ResourceType:    req.ResourceType,
		ResourceID:      req.ResourceID,
		ChargingMode:    req.ChargingMode,
		Currency:        req.Currency,
		IdempotencyKey:  req.IdempotencyKey,
		RequestHash:     req.RequestHash,
		PricingSnapshot: req.PricingSnapshot,
	}

	return r.queries.CreateBillingCharge(
		ctx,
		params,
	)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Charge, error) {
	params := sqlc.GetBillingChargeParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetBillingCharge(
		ctx,
		params,
	)
}

func (r *Repository) ByIdempotencyKey(
	ctx context.Context,
	organizationID uuid.UUID,
	key string,
) (sqlc.Charge, error) {
	params := sqlc.GetBillingChargeByIdempotencyKeyParams{
		OrganizationID: organizationID,
		IdempotencyKey: key,
	}

	return r.queries.GetBillingChargeByIdempotencyKey(
		ctx,
		params,
	)
}

func (r *Repository) ByResource(
	ctx context.Context,
	organizationID uuid.UUID,
	resourceType string,
	resourceID uuid.UUID,
) (sqlc.Charge, error) {
	params := sqlc.GetBillingChargeByResourceParams{
		OrganizationID: organizationID,
		ResourceType:   resourceType,
		ResourceID:     resourceID,
	}

	return r.queries.GetBillingChargeByResource(
		ctx,
		params,
	)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	req ListRequest,
) ([]sqlc.Charge, error) {
	params := sqlc.ListBillingChargesParams{
		OrganizationID: organizationID,
		Status:         req.Status,
		PageLimit:      req.Limit,
		PageOffset:     req.Offset,
	}

	return r.queries.ListBillingCharges(
		ctx,
		params,
	)
}

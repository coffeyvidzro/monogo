package wholesale

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) CreateProviderCDR(
	ctx context.Context,
	params sqlc.CreateProviderCDRParams,
) (sqlc.ProviderCdr, error) {
	return r.queries.CreateProviderCDR(
		ctx,
		params,
	)
}

func (r *Repository) GetProviderCDR(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.ProviderCdr, error) {
	return r.queries.GetProviderCDRByID(
		ctx,
		id,
	)
}

func (r *Repository) GetProviderCDRByProviderRecord(
	ctx context.Context,
	providerID uuid.UUID,
	providerCDRID string,
) (sqlc.ProviderCdr, error) {
	params := sqlc.GetProviderCDRByProviderRecordParams{
		ProviderID:    providerID,
		ProviderCdrID: providerCDRID,
	}

	return r.queries.GetProviderCDRByProviderRecord(
		ctx,
		params,
	)
}

func (r *Repository) ListProviderCDRsByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]sqlc.ProviderCdr, error) {
	return r.queries.ListProviderCDRsByCall(
		ctx,
		callID,
	)
}

func (r *Repository) CreateCharge(
	ctx context.Context,
	params sqlc.CreateWholesaleChargeParams,
) (sqlc.WholesaleCharge, error) {
	return r.queries.CreateWholesaleCharge(
		ctx,
		params,
	)
}

func (r *Repository) GetCharge(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.WholesaleCharge, error) {
	return r.queries.GetWholesaleChargeByID(
		ctx,
		id,
	)
}

func (r *Repository) GetChargeByProviderCDR(
	ctx context.Context,
	providerCDRID uuid.UUID,
) (sqlc.WholesaleCharge, error) {
	return r.queries.GetWholesaleChargeByProviderCDR(
		ctx,
		providerCDRID,
	)
}

func (r *Repository) ListChargesByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]sqlc.WholesaleCharge, error) {
	return r.queries.ListWholesaleChargesByCall(
		ctx,
		callID,
	)
}

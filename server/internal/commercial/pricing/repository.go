package pricing

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

func (r *Repository) Create(
	ctx context.Context,
	params sqlc.CreateCarrierRateParams,
) (sqlc.CarrierRate, error) {
	return r.queries.CreateCarrierRate(
		ctx,
		params,
	)
}

func (r *Repository) Get(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.CarrierRate, error) {
	return r.queries.GetCarrierRateByID(
		ctx,
		id,
	)
}

func (r *Repository) Resolve(
	ctx context.Context,
	params sqlc.ResolveCarrierRateParams,
) (sqlc.CarrierRate, error) {
	return r.queries.ResolveCarrierRate(
		ctx,
		params,
	)
}

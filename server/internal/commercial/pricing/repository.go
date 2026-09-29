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

func (r *Repository) CreateVoiceRate(
	ctx context.Context,
	params sqlc.CreateVoiceRateParams,
) (sqlc.VoiceRate, error) {
	return r.queries.CreateVoiceRate(
		ctx,
		params,
	)
}

func (r *Repository) GetVoiceRate(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.VoiceRate, error) {
	return r.queries.GetVoiceRateByID(
		ctx,
		id,
	)
}

func (r *Repository) ResolveVoiceRate(
	ctx context.Context,
	params sqlc.ResolveVoiceRateParams,
) (sqlc.VoiceRate, error) {
	return r.queries.ResolveVoiceRate(
		ctx,
		params,
	)
}

func (r *Repository) ResolveProductRate(
	ctx context.Context,
	params sqlc.ResolveProductRateParams,
) (sqlc.ProductRate, error) {
	return r.queries.ResolveProductRate(
		ctx,
		params,
	)
}

package plans

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
	params sqlc.CreateSubscriptionPlanParams,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.CreateSubscriptionPlan(
		ctx,
		params,
	)
}

func (r *Repository) GetByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetActiveSubscriptionPlanByID(
		ctx,
		id,
	)
}

func (r *Repository) GetByCode(
	ctx context.Context,
	code string,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetActiveSubscriptionPlanByCode(
		ctx,
		code,
	)
}

func (r *Repository) List(
	ctx context.Context,
) ([]sqlc.SubscriptionPlan, error) {
	return r.queries.ListActiveSubscriptionPlans(ctx)
}

func (r *Repository) Archive(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.ArchiveSubscriptionPlan(
		ctx,
		id,
	)
}

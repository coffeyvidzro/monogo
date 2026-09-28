package subscriptions

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

func (r *Repository) CreatePlan(
	ctx context.Context,
	params sqlc.CreateSubscriptionPlanParams,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.CreateSubscriptionPlan(
		ctx,
		params,
	)
}

func (r *Repository) GetPlanByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetActiveSubscriptionPlanByID(
		ctx,
		id,
	)
}

func (r *Repository) GetPlanByCode(
	ctx context.Context,
	code string,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetActiveSubscriptionPlanByCode(
		ctx,
		code,
	)
}

func (r *Repository) ListPlans(
	ctx context.Context,
) ([]sqlc.SubscriptionPlan, error) {
	return r.queries.ListActiveSubscriptionPlans(ctx)
}

func (r *Repository) ArchivePlan(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.ArchiveSubscriptionPlan(
		ctx,
		id,
	)
}

func (r *Repository) CreateSubscription(
	ctx context.Context,
	organizationID uuid.UUID,
	planID uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.CreateSubscriptionParams{
		PlanID:         planID,
		OrganizationID: organizationID,
	}

	return r.queries.CreateSubscription(
		ctx,
		params,
	)
}

func (r *Repository) GetSubscription(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.GetSubscriptionByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetSubscriptionByID(
		ctx,
		params,
	)
}

func (r *Repository) GetCurrent(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.GetCurrentSubscriptionByOrganization(
		ctx,
		organizationID,
	)
}

func (r *Repository) Activate(
	ctx context.Context,
	params sqlc.ActivateSubscriptionParams,
) (sqlc.Subscription, error) {
	return r.queries.ActivateSubscription(
		ctx,
		params,
	)
}

func (r *Repository) MarkPastDue(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.MarkSubscriptionPastDueParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.MarkSubscriptionPastDue(
		ctx,
		params,
	)
}

func (r *Repository) SetCancelAtPeriodEnd(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.SetSubscriptionCancelAtPeriodEndParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.SetSubscriptionCancelAtPeriodEnd(
		ctx,
		params,
	)
}

func (r *Repository) Cancel(
	ctx context.Context,
	params sqlc.CancelSubscriptionParams,
) (sqlc.Subscription, error) {
	return r.queries.CancelSubscription(
		ctx,
		params,
	)
}

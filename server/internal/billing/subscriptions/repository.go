package subscriptions

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	if queries == nil {
		panic("billing subscriptions: queries are required")
	}

	return &Repository{
		queries: queries,
	}
}

func (r *Repository) CreatePlan(
	ctx context.Context,
	req CreatePlanRequest,
) (sqlc.SubscriptionPlan, error) {
	params := sqlc.CreateSubscriptionPlanParams{
		Code:         req.Code,
		Name:         req.Name,
		Currency:     req.Currency,
		AmountMicros: req.AmountMicros,
	}

	return r.queries.CreateSubscriptionPlan(
		ctx,
		params,
	)
}

func (r *Repository) GetPlanByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetSubscriptionPlanByID(
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

func (r *Repository) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Subscription, error) {
	params := sqlc.CreateSubscriptionParams{
		OrganizationID: req.OrganizationID,
		PlanID:         req.PlanID,
	}

	return r.queries.CreateSubscription(
		ctx,
		params,
	)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.GetSubscriptionParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetSubscription(
		ctx,
		params,
	)
}

func (r *Repository) Current(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.GetCurrentSubscription(
		ctx,
		organizationID,
	)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Subscription, error) {
	return r.queries.ListSubscriptions(
		ctx,
		organizationID,
	)
}

func (r *Repository) Activate(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	params := sqlc.ActivateSubscriptionParams{
		PeriodStart: pgconv.TimeToTimestamptz(
			req.Start,
		),
		PeriodEnd: pgconv.TimeToTimestamptz(
			req.End,
		),
		ID:             req.SubscriptionID,
		OrganizationID: req.OrganizationID,
	}

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
	value bool,
) (sqlc.Subscription, error) {
	params := sqlc.SetSubscriptionCancelAtPeriodEndParams{
		CancelAtPeriodEnd: value,
		ID:                id,
		OrganizationID:    organizationID,
	}

	return r.queries.SetSubscriptionCancelAtPeriodEnd(
		ctx,
		params,
	)
}

func (r *Repository) Renew(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	params := sqlc.RenewSubscriptionParams{
		PeriodStart: pgconv.TimeToTimestamptz(
			req.Start,
		),
		PeriodEnd: pgconv.TimeToTimestamptz(
			req.End,
		),
		ID:             req.SubscriptionID,
		OrganizationID: req.OrganizationID,
	}

	return r.queries.RenewSubscription(
		ctx,
		params,
	)
}

func (r *Repository) Cancel(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscription, error) {
	params := sqlc.CancelSubscriptionParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.CancelSubscription(
		ctx,
		params,
	)
}

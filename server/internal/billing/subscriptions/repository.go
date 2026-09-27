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
	return &Repository{queries: queries}
}

func (r *Repository) CreatePlan(
	ctx context.Context,
	req CreatePlanRequest,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.CreateSubscriptionPlan(ctx, sqlc.CreateSubscriptionPlanParams{
		Code:         req.Code,
		Name:         req.Name,
		Currency:     req.Currency,
		AmountMicros: req.AmountMicros,
	})
}

func (r *Repository) GetPlanByID(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetSubscriptionPlanByID(ctx, id)
}

func (r *Repository) GetPlanByCode(
	ctx context.Context,
	code string,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.GetActiveSubscriptionPlanByCode(ctx, code)
}

func (r *Repository) ListPlans(ctx context.Context) ([]sqlc.SubscriptionPlan, error) {
	return r.queries.ListActiveSubscriptionPlans(ctx)
}

func (r *Repository) ArchivePlan(
	ctx context.Context,
	id uuid.UUID,
) (sqlc.SubscriptionPlan, error) {
	return r.queries.ArchiveSubscriptionPlan(ctx, id)
}

func (r *Repository) Create(
	ctx context.Context,
	req CreateRequest,
) (sqlc.Subscription, error) {
	return r.queries.CreateSubscription(ctx, sqlc.CreateSubscriptionParams{
		OrganizationID: req.OrganizationID,
		PlanID:         req.PlanID,
	})
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.GetSubscription(ctx, sqlc.GetSubscriptionParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Current(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.GetCurrentSubscription(ctx, organizationID)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Subscription, error) {
	return r.queries.ListSubscriptions(ctx, organizationID)
}

func (r *Repository) Activate(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	return r.queries.ActivateSubscription(ctx, sqlc.ActivateSubscriptionParams{
		PeriodStart:    pgconv.TimeToTimestamptz(req.Start),
		PeriodEnd:      pgconv.TimeToTimestamptz(req.End),
		ID:             req.SubscriptionID,
		OrganizationID: req.OrganizationID,
	})
}

func (r *Repository) MarkPastDue(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.MarkSubscriptionPastDue(ctx, sqlc.MarkSubscriptionPastDueParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) SetCancelAtPeriodEnd(
	ctx context.Context,
	organizationID, id uuid.UUID,
	value bool,
) (sqlc.Subscription, error) {
	return r.queries.SetSubscriptionCancelAtPeriodEnd(
		ctx,
		sqlc.SetSubscriptionCancelAtPeriodEndParams{
			CancelAtPeriodEnd: value,
			ID:                id,
			OrganizationID:    organizationID,
		},
	)
}

func (r *Repository) Renew(
	ctx context.Context,
	req PeriodRequest,
) (sqlc.Subscription, error) {
	return r.queries.RenewSubscription(ctx, sqlc.RenewSubscriptionParams{
		PeriodStart:    pgconv.TimeToTimestamptz(req.Start),
		PeriodEnd:      pgconv.TimeToTimestamptz(req.End),
		ID:             req.SubscriptionID,
		OrganizationID: req.OrganizationID,
	})
}

func (r *Repository) Cancel(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Subscription, error) {
	return r.queries.CancelSubscription(ctx, sqlc.CancelSubscriptionParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

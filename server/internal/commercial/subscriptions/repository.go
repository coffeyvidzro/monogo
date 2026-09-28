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

func (r *Repository) List(
	ctx context.Context,
	req ListRequest,
) ([]sqlc.Subscription, error) {
	params := sqlc.ListSubscriptionsByOrganizationParams{
		OrganizationID: req.OrganizationID,
		OffsetCount:    req.Offset,
		LimitCount:     req.Limit,
	}

	return r.queries.ListSubscriptionsByOrganization(
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

func (r *Repository) UpdateCancelAtPeriodEnd(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	cancelAtPeriodEnd bool,
) (sqlc.Subscription, error) {
	params := sqlc.UpdateSubscriptionCancelAtPeriodEndParams{
		CancelAtPeriodEnd: cancelAtPeriodEnd,
		ID:                id,
		OrganizationID:    organizationID,
	}

	return r.queries.UpdateSubscriptionCancelAtPeriodEnd(
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

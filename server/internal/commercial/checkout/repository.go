package checkout

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	purpose string,
	subscriptionID *uuid.UUID,
	reference string,
	amountMicros int64,
	currency string,
	periodStart *time.Time,
	periodEnd *time.Time,
	expiresAt time.Time,
) (sqlc.Checkout, error) {
	var start pgtype.Timestamptz
	if periodStart != nil {
		start = pgconv.TimeToTimestamptz(*periodStart)
	}

	var end pgtype.Timestamptz
	if periodEnd != nil {
		end = pgconv.TimeToTimestamptz(*periodEnd)
	}

	return r.queries.CreateCheckout(
		ctx,
		sqlc.CreateCheckoutParams{
			OrganizationID: organizationID,
			Purpose:        purpose,
			SubscriptionID: subscriptionID,
			Reference:      reference,
			AmountMicros:   amountMicros,
			Currency:       currency,
			PeriodStart:    start,
			PeriodEnd:      end,
			ExpiresAt:      pgconv.TimeToTimestamptz(expiresAt),
		},
	)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Checkout, error) {
	return r.queries.GetCheckoutByID(
		ctx,
		sqlc.GetCheckoutByIDParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) Confirm(
	ctx context.Context,
	req ConfirmRequest,
	now time.Time,
) (sqlc.Checkout, error) {
	return r.queries.ConfirmCheckout(
		ctx,
		sqlc.ConfirmCheckoutParams{
			Provider:       &req.Provider,
			PaymentMethod:  &req.PaymentMethod,
			ID:             req.CheckoutID,
			OrganizationID: req.OrganizationID,
			NowAt:          pgconv.TimeToTimestamptz(now),
		},
	)
}

func (r *Repository) GetForContinuation(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	now time.Time,
) (sqlc.Checkout, error) {
	return r.queries.GetCheckoutForContinuation(
		ctx,
		sqlc.GetCheckoutForContinuationParams{
			ID:             id,
			OrganizationID: organizationID,
			NowAt:          pgconv.TimeToTimestamptz(now),
		},
	)
}


func (r *Repository) Complete(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	completedAt time.Time,
) (sqlc.Checkout, error) {
	return r.queries.CompleteCheckout(
		ctx,
		sqlc.CompleteCheckoutParams{
			CompletedAt:    pgconv.TimeToTimestamptz(completedAt),
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) Fail(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	providerMessage *string,
	completedAt time.Time,
) (sqlc.Checkout, error) {
	return r.queries.FailCheckout(
		ctx,
		sqlc.FailCheckoutParams{
			ProviderMessage: providerMessage,
			CompletedAt:     pgconv.TimeToTimestamptz(completedAt),
			ID:              id,
			OrganizationID:  organizationID,
		},
	)
}

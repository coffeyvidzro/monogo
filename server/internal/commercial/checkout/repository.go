package checkout

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		queries: r.queries.WithTx(tx),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	purpose string,
	subscriptionID *uuid.UUID,
	reference string,
	amountMicros int64,
	currency string,
	expiresAt time.Time,
) (sqlc.Checkout, error) {

	return r.queries.CreateCheckout(
		ctx,
		sqlc.CreateCheckoutParams{
			OrganizationID: organizationID,
			Purpose:        purpose,
			SubscriptionID: subscriptionID,
			Reference:      reference,
			AmountMicros:   amountMicros,
			Currency:       currency,
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
			CheckoutID:     id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) Confirm(
	ctx context.Context,
	req ConfirmRequest,
	now time.Time,
) (sqlc.Checkout, error) {
	row, err := r.queries.ConfirmCheckout(
		ctx,
		sqlc.ConfirmCheckoutParams{
			Provider:       &req.Provider,
			PaymentMethod:  &req.PaymentMethod,
			CheckoutID:     req.CheckoutID,
			OrganizationID: req.OrganizationID,
			NowAt:          pgconv.TimeToTimestamptz(now),
		},
	)
	if err != nil {
		return sqlc.Checkout{}, err
	}

	return sqlc.Checkout(row), nil
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
			CheckoutID:     id,
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
			CheckoutID:     id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) Fail(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	failureCode string,
	providerMessage *string,
	completedAt time.Time,
) (sqlc.Checkout, error) {
	return r.queries.FailCheckout(
		ctx,
		sqlc.FailCheckoutParams{
			ProviderMessage: providerMessage,
			FailureCode:     &failureCode,
			CompletedAt:     pgconv.TimeToTimestamptz(completedAt),
			CheckoutID:      id,
			OrganizationID:  organizationID,
		},
	)
}

func (r *Repository) ExpireDue(
	ctx context.Context,
	completedAt time.Time,
	limit int32,
) (int64, error) {
	return r.queries.ExpireDueCheckouts(
		ctx,
		sqlc.ExpireDueCheckoutsParams{
			CompletedAt: pgconv.TimeToTimestamptz(completedAt),
			LimitCount:  limit,
		},
	)
}

func (r *Repository) UpdateAction(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	nextAction string,
	providerMessage *string,
) (sqlc.Checkout, error) {
	return r.queries.UpdateCheckoutAction(
		ctx,
		sqlc.UpdateCheckoutActionParams{
			NextAction:      nextAction,
			ProviderMessage: providerMessage,
			CheckoutID:      id,
			OrganizationID:  organizationID,
		},
	)
}

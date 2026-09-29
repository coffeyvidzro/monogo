package payments

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
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

func (r *Repository) CreateAttempt(
	ctx context.Context,
	req CreateAttemptRequest,
	attempt int32,
) (sqlc.Payment, error) {
	return r.queries.CreatePaymentAttempt(
		ctx,
		sqlc.CreatePaymentAttemptParams{
			CheckoutID:     req.CheckoutID,
			OrganizationID: req.OrganizationID,
			Provider:       req.Provider,
			Attempt:        attempt,
			AmountMicros:   req.AmountMicros,
			Currency:       req.Currency,
		},
	)
}

func (r *Repository) GetActiveByCheckout(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
) (sqlc.Payment, error) {
	return r.queries.GetActivePaymentAttemptByCheckout(
		ctx,
		sqlc.GetActivePaymentAttemptByCheckoutParams{
			CheckoutID:     checkoutID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) NextAttempt(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
) (int64, error) {
	return r.queries.GetNextPaymentAttemptNumber(
		ctx,
		sqlc.GetNextPaymentAttemptNumberParams{
			CheckoutID:     checkoutID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) AttachProviderPaymentID(
	ctx context.Context,
	req AttachProviderPaymentIDRequest,
) (sqlc.Payment, error) {
	return r.queries.AttachProviderPaymentID(
		ctx,
		sqlc.AttachProviderPaymentIDParams{
			ProviderPaymentID: &req.ProviderPaymentID,
			ID:                req.PaymentID,
			CheckoutID:        req.CheckoutID,
			OrganizationID:    req.OrganizationID,
		},
	)
}

func (r *Repository) MarkSucceeded(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
	id uuid.UUID,
	completedAt time.Time,
) (sqlc.Payment, error) {
	return r.queries.MarkPaymentAttemptSucceeded(
		ctx,
		sqlc.MarkPaymentAttemptSucceededParams{
			PaidAt:         pgconv.TimeToTimestamptz(completedAt),
			ID:             id,
			CheckoutID:     checkoutID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	organizationID uuid.UUID,
	checkoutID uuid.UUID,
	id uuid.UUID,
	failureCode string,
	completedAt time.Time,
) (sqlc.Payment, error) {
	return r.queries.MarkPaymentAttemptFailed(
		ctx,
		sqlc.MarkPaymentAttemptFailedParams{
			FailureCode:    &failureCode,
			ID:             id,
			CheckoutID:     checkoutID,
			OrganizationID: organizationID,
		},
	)
}

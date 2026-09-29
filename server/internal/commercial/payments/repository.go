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
			PaymentMethod:  req.PaymentMethod,
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
	nextAttempt, err := r.queries.GetNextPaymentAttemptNumber(
		ctx,
		sqlc.GetNextPaymentAttemptNumberParams{
			CheckoutID:     checkoutID,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return 0, err
	}

	return int64(nextAttempt), nil
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

func (r *Repository) GetByProviderPaymentID(
	ctx context.Context,
	provider string,
	providerPaymentID string,
) (sqlc.Payment, error) {
	return r.queries.GetPaymentAttemptByProviderPaymentID(
		ctx,
		sqlc.GetPaymentAttemptByProviderPaymentIDParams{
			Provider:          provider,
			ProviderPaymentID: &providerPaymentID,
		},
	)
}

func (r *Repository) CreateProviderEvent(
	ctx context.Context,
	payment sqlc.Payment,
	req RecordProviderEventRequest,
	payloadSHA256 string,
	receivedAt time.Time,
) (sqlc.PaymentProviderEvent, error) {
	return r.queries.CreatePaymentProviderEvent(
		ctx,
		sqlc.CreatePaymentProviderEventParams{
			PaymentID:       payment.ID,
			OrganizationID:  payment.OrganizationID,
			Provider:        req.Provider,
			ProviderEventID: req.ProviderEventID,
			EventType:       req.EventType,
			PayloadSha256:   payloadSHA256,
			Payload:         req.Payload,
			ReceivedAt:      pgconv.TimeToTimestamptz(receivedAt),
		},
	)
}

func (r *Repository) GetProviderEventByIdentity(
	ctx context.Context,
	provider string,
	providerEventID string,
) (sqlc.PaymentProviderEvent, error) {
	return r.queries.GetPaymentProviderEventByIdentity(
		ctx,
		sqlc.GetPaymentProviderEventByIdentityParams{
			Provider:        provider,
			ProviderEventID: providerEventID,
		},
	)
}

func (r *Repository) MarkProviderEventProcessed(
	ctx context.Context,
	id uuid.UUID,
	processedAt time.Time,
) (sqlc.PaymentProviderEvent, error) {
	return r.queries.MarkPaymentProviderEventProcessed(
		ctx,
		sqlc.MarkPaymentProviderEventProcessedParams{
			ProcessedAt: pgconv.TimeToTimestamptz(processedAt),
			ID:          id,
		},
	)
}

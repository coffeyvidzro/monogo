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

func (r *Repository) CreateSubscription(
	ctx context.Context,
	organizationID uuid.UUID,
	subscriptionID uuid.UUID,
	provider string,
	amountMicros int64,
	currency string,
	periodStart time.Time,
	periodEnd time.Time,
) (sqlc.Payment, error) {
	return r.queries.CreateSubscriptionPayment(
		ctx,
		sqlc.CreateSubscriptionPaymentParams{
			OrganizationID: organizationID,
			Provider:       provider,
			SubscriptionID: subscriptionID,
			AmountMicros:   amountMicros,
			Currency:       currency,
			PeriodStart:    pgconv.TimeToTimestamptz(periodStart),
			PeriodEnd:      pgconv.TimeToTimestamptz(periodEnd),
		},
	)
}

func (r *Repository) CreateWalletTopup(
	ctx context.Context,
	organizationID uuid.UUID,
	provider string,
	amountMicros int64,
	currency string,
) (sqlc.Payment, error) {
	return r.queries.CreateWalletTopupPayment(
		ctx,
		sqlc.CreateWalletTopupPaymentParams{
			OrganizationID: organizationID,
			Provider:       provider,
			AmountMicros:   amountMicros,
			Currency:       currency,
		},
	)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Payment, error) {
	return r.queries.GetPaymentByID(
		ctx,
		sqlc.GetPaymentByIDParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) AttachProviderReference(
	ctx context.Context,
	req AttachProviderReferenceRequest,
) (sqlc.Payment, error) {
	return r.queries.AttachPaymentProviderReference(
		ctx,
		sqlc.AttachPaymentProviderReferenceParams{
			ProviderReference: req.ProviderReference,
			ID:                req.PaymentID,
			OrganizationID:    req.OrganizationID,
		},
	)
}

func (r *Repository) ClaimProviderEvent(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	providerEventID string,
) (sqlc.Payment, error) {
	return r.queries.ClaimPaymentProviderEvent(
		ctx,
		sqlc.ClaimPaymentProviderEventParams{
			ProviderEventID: providerEventID,
			ID:              id,
			OrganizationID:  organizationID,
		},
	)
}

func (r *Repository) MarkSucceeded(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	providerEventID string,
	completedAt time.Time,
) (sqlc.Payment, error) {
	return r.queries.MarkPaymentSucceeded(
		ctx,
		sqlc.MarkPaymentSucceededParams{
			CompletedAt:     pgconv.TimeToTimestamptz(completedAt),
			ID:              id,
			OrganizationID:  organizationID,
			ProviderEventID: providerEventID,
		},
	)
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	req FailRequest,
) (sqlc.Payment, error) {
	return r.queries.MarkPaymentFailed(
		ctx,
		sqlc.MarkPaymentFailedParams{
			FailureCode:     req.FailureCode,
			CompletedAt:     pgconv.TimeToTimestamptz(req.OccurredAt),
			ID:              req.PaymentID,
			OrganizationID:  req.OrganizationID,
			ProviderEventID: req.ProviderEventID,
		},
	)
}

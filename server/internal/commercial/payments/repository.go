package payments

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		return &Repository{}
	}
	return &Repository{queries: sqlc.New(db)}
}

func (r *Repository) Available() bool {
	return r != nil && r.queries != nil
}

func (r *Repository) Create(ctx context.Context, req Attempt) (sqlc.Payment, error) {
	return r.queries.CreateCheckoutPayment(ctx, sqlc.CreateCheckoutPaymentParams{
		OrganizationID: req.OrganizationID,
		CheckoutID:     req.CheckoutID,
		Provider:       req.Provider,
		AttemptKey:     req.AttemptKey,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
	})
}

func (r *Repository) ByAttempt(ctx context.Context, req Attempt) (sqlc.Payment, error) {
	return r.queries.GetCheckoutPaymentByAttempt(ctx, sqlc.GetCheckoutPaymentByAttemptParams{
		OrganizationID: req.OrganizationID,
		CheckoutID:     req.CheckoutID,
		Provider:       req.Provider,
		AttemptKey:     req.AttemptKey,
	})
}

func (r *Repository) Get(ctx context.Context, organizationID, paymentID uuid.UUID) (sqlc.Payment, error) {
	return r.queries.GetCheckoutPayment(ctx, sqlc.GetCheckoutPaymentParams{
		OrganizationID: organizationID,
		ID:             paymentID,
	})
}

func (r *Repository) RecordEvent(ctx context.Context, event Event) (sqlc.PaymentEvent, error) {
	return r.queries.RecordIncomingPaymentEvent(ctx, sqlc.RecordIncomingPaymentEventParams{
		PaymentID:       event.PaymentID,
		Provider:        event.Provider,
		ProviderEventID: event.ProviderEventID,
		EventType:       event.EventType,
		PayloadSha256:   event.PayloadSHA256,
	})
}

func (r *Repository) EventByIdentity(ctx context.Context, event Event) (sqlc.PaymentEvent, error) {
	return r.queries.GetIncomingPaymentEventByIdentity(ctx, sqlc.GetIncomingPaymentEventByIdentityParams{
		Provider:        event.Provider,
		ProviderEventID: event.ProviderEventID,
	})
}

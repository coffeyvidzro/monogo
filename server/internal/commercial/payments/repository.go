package payments

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	if db == nil {
		return &Repository{}
	}
	return &Repository{db: db, queries: sqlc.New(db)}
}

func (r *Repository) Available() bool {
	return r != nil && r.queries != nil
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) { return r.db.Begin(ctx) }

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

func (r *Repository) Lock(ctx context.Context, organizationID, paymentID uuid.UUID) (sqlc.Payment, error) {
	return r.queries.LockCheckoutPayment(ctx, sqlc.LockCheckoutPaymentParams{OrganizationID: organizationID, ID: paymentID})
}

func (r *Repository) LockCheckout(ctx context.Context, id uuid.UUID) (sqlc.Checkout, error) {
	return r.queries.LockWalletCheckout(ctx, id)
}

func (r *Repository) Succeed(ctx context.Context, id uuid.UUID, reference string, verifiedAt time.Time) (sqlc.Payment, error) {
	return r.queries.MarkCheckoutPaymentSucceeded(ctx, sqlc.MarkCheckoutPaymentSucceededParams{ID: id, ProviderReference: reference, VerifiedAt: pgconv.TimeToTimestamptz(verifiedAt)})
}

func (r *Repository) LinkTransaction(ctx context.Context, id, transactionID uuid.UUID) (sqlc.Payment, error) {
	return r.queries.LinkPaymentWalletTransaction(ctx, sqlc.LinkPaymentWalletTransactionParams{ID: id, WalletTransactionID: transactionID})
}

func (r *Repository) CompleteCheckout(
	ctx context.Context,
	current sqlc.Checkout,
	payment sqlc.Payment,
	entry sqlc.WalletTransaction,
) (sqlc.Checkout, error) {
	return checkout.CompleteWithCredit(ctx, r.queries, current, payment, entry)
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

func (r *Repository) Checkout(ctx context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	return r.queries.GetPaymentCheckout(ctx, sqlc.GetPaymentCheckoutParams{OrganizationID: organizationID, ID: checkoutID})
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

func (r *Repository) ListRecovery(ctx context.Context, limit int32) ([]RecoveryCandidate, error) {
	rows, err := r.queries.ListPaymentsDueForRecovery(ctx, limit)
	if err != nil {
		return nil, err
	}
	result := make([]RecoveryCandidate, 0, len(rows))
	for _, row := range rows {
		if row.ProviderReference == nil {
			continue
		}
		result = append(result, RecoveryCandidate{OrganizationID: row.OrganizationID, PaymentID: row.ID,
			Provider: row.Provider, ProviderReference: *row.ProviderReference, AmountMinor: row.AmountMinor, Currency: row.Currency})
	}
	return result, nil
}

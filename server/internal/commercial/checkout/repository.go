package checkout

import (
	"context"

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

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) { return r.db.Begin(ctx) }

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

func (r *Repository) Available() bool {
	return r != nil && r.queries != nil
}

func (r *Repository) Create(ctx context.Context, req CreateRequest, hash string) (sqlc.Checkout, error) {
	return r.queries.CreateWalletCheckout(ctx, sqlc.CreateWalletCheckoutParams{
		OrganizationID: req.OrganizationID,
		WalletID:       req.WalletID,
		AmountMinor:    req.AmountMinor,
		Currency:       req.Currency,
		IdempotencyKey: req.IdempotencyKey,
		RequestHash:    hash,
		ExpiresAt:      pgconv.TimeToTimestamptz(req.ExpiresAt),
	})
}

func (r *Repository) ByKey(ctx context.Context, req CreateRequest) (sqlc.Checkout, error) {
	return r.queries.GetWalletCheckoutByKey(ctx, sqlc.GetWalletCheckoutByKeyParams{
		OrganizationID: req.OrganizationID,
		WalletID:       req.WalletID,
		IdempotencyKey: req.IdempotencyKey,
	})
}

func (r *Repository) Get(ctx context.Context, organizationID, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	return r.queries.GetWalletCheckout(ctx, sqlc.GetWalletCheckoutParams{
		OrganizationID: organizationID,
		ID:             checkoutID,
	})
}

func (r *Repository) Cancel(ctx context.Context, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	return r.queries.CancelWalletCheckout(ctx, checkoutID)
}

func (r *Repository) Lock(ctx context.Context, checkoutID uuid.UUID) (sqlc.Checkout, error) {
	return r.queries.LockWalletCheckout(ctx, checkoutID)
}

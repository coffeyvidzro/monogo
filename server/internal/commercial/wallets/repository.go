package wallets

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

// Repository owns the wallet persistence queries. The service coordinates
// transaction boundaries so a balance update and ledger entry commit together.
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
	return r != nil && r.db != nil && r.queries != nil
}

func (r *Repository) Begin(ctx context.Context) (pgx.Tx, error) {
	return r.db.Begin(ctx)
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

func (r *Repository) LockByID(ctx context.Context, organizationID, walletID uuid.UUID) (sqlc.Wallet, error) {
	return r.queries.LockPrepaidWalletByID(ctx, sqlc.LockPrepaidWalletByIDParams{WalletID: walletID, OrganizationID: organizationID})
}

func (r *Repository) Transaction(ctx context.Context, id uuid.UUID) (sqlc.WalletTransaction, error) {
	return r.queries.GetWalletTransaction(ctx, id)
}

func (r *Repository) Create(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	return r.queries.CreatePrepaidWallet(ctx, sqlc.CreatePrepaidWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
}

func (r *Repository) Get(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	return r.queries.GetPrepaidWallet(ctx, sqlc.GetPrepaidWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
}

func (r *Repository) Lock(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	return r.queries.LockPrepaidWallet(ctx, sqlc.LockPrepaidWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
}

func (r *Repository) FindTransaction(
	ctx context.Context,
	walletID uuid.UUID,
	referenceType string,
	referenceID uuid.UUID,
) (sqlc.WalletTransaction, error) {
	return r.queries.GetWalletTransactionByReference(ctx, sqlc.GetWalletTransactionByReferenceParams{
		WalletID:      walletID,
		ReferenceType: referenceType,
		ReferenceID:   referenceID,
	})
}

func (r *Repository) SetBalance(
	ctx context.Context,
	wallet sqlc.Wallet,
	organizationID uuid.UUID,
	next int64,
) (sqlc.Wallet, error) {
	return r.queries.SetPrepaidWalletBalance(ctx, sqlc.SetPrepaidWalletBalanceParams{
		WalletID:             wallet.ID,
		OrganizationID:       organizationID,
		PreviousBalanceMinor: wallet.BalanceMinor,
		BalanceMinor:         next,
	})
}

func (r *Repository) Record(
	ctx context.Context,
	walletID uuid.UUID,
	entry Entry,
	next int64,
) (sqlc.WalletTransaction, error) {
	return r.queries.CreateWalletTransaction(ctx, sqlc.CreateWalletTransactionParams{
		WalletID:          walletID,
		Direction:         entry.Direction,
		Reason:            entry.Reason,
		AmountMinor:       entry.AmountMinor,
		BalanceAfterMinor: next,
		ReferenceType:     entry.ReferenceType,
		ReferenceID:       entry.ReferenceID,
	})
}

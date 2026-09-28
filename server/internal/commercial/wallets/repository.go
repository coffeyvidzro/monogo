package wallets

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{
		queries: r.queries.WithTx(tx),
	}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.CreateWallet(
		ctx,
		organizationID,
	)
}

func (r *Repository) GetActive(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.GetActiveWalletByOrganization(
		ctx,
		organizationID,
	)
}

func (r *Repository) LockActive(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.LockActiveWallet(
		ctx,
		organizationID,
	)
}

func (r *Repository) ApplyBalance(
	ctx context.Context,
	organizationID uuid.UUID,
	deltaMicros int64,
) (sqlc.Wallet, error) {
	params := sqlc.ApplyWalletBalanceParams{
		DeltaMicros:    deltaMicros,
		OrganizationID: organizationID,
	}

	return r.queries.ApplyWalletBalance(
		ctx,
		params,
	)
}

func (r *Repository) CreateLedgerEntry(
	ctx context.Context,
	params sqlc.CreateWalletLedgerEntryParams,
) (sqlc.WalletLedgerEntry, error) {
	return r.queries.CreateWalletLedgerEntry(
		ctx,
		params,
	)
}

func (r *Repository) GetLedgerEntryByOperation(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	params := sqlc.GetWalletLedgerEntryByOperationIDParams{
		OrganizationID: organizationID,
		OperationID:    operationID,
	}

	return r.queries.GetWalletLedgerEntryByOperationID(
		ctx,
		params,
	)
}

func (r *Repository) ListLedgerEntries(
	ctx context.Context,
	req ListLedgerRequest,
) ([]sqlc.WalletLedgerEntry, error) {
	params := sqlc.ListWalletLedgerEntriesParams{
		OrganizationID: req.OrganizationID,
		OffsetCount:    req.Offset,
		LimitCount:     req.Limit,
	}

	return r.queries.ListWalletLedgerEntries(
		ctx,
		params,
	)
}

func (r *Repository) Freeze(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.FreezeWallet(
		ctx,
		organizationID,
	)
}

func (r *Repository) Activate(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.ActivateWallet(
		ctx,
		organizationID,
	)
}

func (r *Repository) Close(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.CloseWallet(
		ctx,
		organizationID,
	)
}

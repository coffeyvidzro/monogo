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
	currency string,
) (sqlc.Wallet, error) {
	params := sqlc.CreateWalletParams{
		Currency:       currency,
		OrganizationID: organizationID,
	}

	return r.queries.CreateWallet(
		ctx,
		params,
	)
}

func (r *Repository) GetActive(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.GetActiveWalletByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.GetActiveWalletByID(
		ctx,
		params,
	)
}

func (r *Repository) GetActiveByCurrency(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	params := sqlc.GetActiveWalletByOrganizationCurrencyParams{
		OrganizationID: organizationID,
		Currency:       currency,
	}

	return r.queries.GetActiveWalletByOrganizationCurrency(
		ctx,
		params,
	)
}

func (r *Repository) LockActive(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.LockActiveWalletParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.LockActiveWallet(
		ctx,
		params,
	)
}

func (r *Repository) ApplyBalance(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	deltaMicros int64,
) (sqlc.Wallet, error) {
	params := sqlc.ApplyWalletBalanceParams{
		DeltaMicros:    deltaMicros,
		ID:             id,
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
	walletID uuid.UUID,
	operationID uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	params := sqlc.GetWalletLedgerEntryByOperationIDParams{
		OrganizationID: organizationID,
		WalletID:       walletID,
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
		WalletID:       req.WalletID,
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
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.FreezeWalletParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.FreezeWallet(
		ctx,
		params,
	)
}

func (r *Repository) Activate(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.ActivateWalletParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.ActivateWallet(
		ctx,
		params,
	)
}

func (r *Repository) Close(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	params := sqlc.CloseWalletParams{
		ID:             id,
		OrganizationID: organizationID,
	}

	return r.queries.CloseWallet(
		ctx,
		params,
	)
}

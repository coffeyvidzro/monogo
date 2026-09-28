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

func (r *Repository) LockForSettlement(
	ctx context.Context,
	organizationID uuid.UUID,
) (sqlc.Wallet, error) {
	return r.queries.LockWalletForSettlement(
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

func (r *Repository) ReserveBalance(
	ctx context.Context,
	organizationID uuid.UUID,
	amountMicros int64,
) (sqlc.Wallet, error) {
	params := sqlc.ReserveWalletBalanceParams{
		AmountMicros:   amountMicros,
		OrganizationID: organizationID,
	}

	return r.queries.ReserveWalletBalance(
		ctx,
		params,
	)
}

func (r *Repository) CaptureReservedBalance(
	ctx context.Context,
	organizationID uuid.UUID,
	amountMicros int64,
) (sqlc.Wallet, error) {
	params := sqlc.CaptureWalletReservedBalanceParams{
		AmountMicros:   amountMicros,
		OrganizationID: organizationID,
	}

	return r.queries.CaptureWalletReservedBalance(
		ctx,
		params,
	)
}

func (r *Repository) ReleaseReservedBalance(
	ctx context.Context,
	organizationID uuid.UUID,
	amountMicros int64,
) (sqlc.Wallet, error) {
	params := sqlc.ReleaseWalletReservedBalanceParams{
		AmountMicros:   amountMicros,
		OrganizationID: organizationID,
	}

	return r.queries.ReleaseWalletReservedBalance(
		ctx,
		params,
	)
}

func (r *Repository) CreateHold(
	ctx context.Context,
	params sqlc.CreateWalletHoldParams,
) (sqlc.WalletHold, error) {
	return r.queries.CreateWalletHold(
		ctx,
		params,
	)
}

func (r *Repository) GetHoldByOperation(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (sqlc.WalletHold, error) {
	params := sqlc.GetWalletHoldByOperationIDParams{
		OrganizationID: organizationID,
		OperationID:    operationID,
	}

	return r.queries.GetWalletHoldByOperationID(
		ctx,
		params,
	)
}

func (r *Repository) MarkHoldCaptured(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (sqlc.WalletHold, error) {
	params := sqlc.MarkWalletHoldCapturedParams{
		OrganizationID: organizationID,
		OperationID:    operationID,
	}

	return r.queries.MarkWalletHoldCaptured(
		ctx,
		params,
	)
}

func (r *Repository) MarkHoldReleased(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (sqlc.WalletHold, error) {
	params := sqlc.MarkWalletHoldReleasedParams{
		OrganizationID: organizationID,
		OperationID:    operationID,
	}

	return r.queries.MarkWalletHoldReleased(
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

package wallets

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
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

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.Wallet, error) {
	return r.queries.ListPrepaidWallets(ctx, organizationID)
}

func (r *Repository) GetByID(ctx context.Context, organizationID, walletID uuid.UUID) (sqlc.Wallet, error) {
	return r.queries.GetPrepaidWalletByID(ctx, sqlc.GetPrepaidWalletByIDParams{
		WalletID: walletID, OrganizationID: organizationID,
	})
}

func (r *Repository) ListTransactions(ctx context.Context, organizationID, walletID uuid.UUID, limit int32) ([]sqlc.WalletTransaction, error) {
	return r.queries.ListWalletTransactionsByWalletID(ctx, sqlc.ListWalletTransactionsByWalletIDParams{
		OrganizationID: organizationID, WalletID: walletID, RowLimit: limit,
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

func (r *Repository) SetAmounts(ctx context.Context, wallet sqlc.Wallet, organizationID uuid.UUID, balance, reserved int64) (sqlc.Wallet, error) {
	return r.queries.SetPrepaidWalletAmounts(ctx, sqlc.SetPrepaidWalletAmountsParams{
		WalletID: wallet.ID, OrganizationID: organizationID, BalanceMinor: balance, ReservedMinor: reserved,
		PreviousBalanceMinor: wallet.BalanceMinor, PreviousReservedMinor: wallet.ReservedMinor,
	})
}

func (r *Repository) CreateReservation(ctx context.Context, walletID uuid.UUID, req ReserveRequest) (sqlc.WalletReservation, error) {
	return r.queries.CreateWalletReservation(ctx, sqlc.CreateWalletReservationParams{WalletID: walletID,
		OrganizationID: req.OrganizationID, AmountMinor: req.AmountMinor, OperationType: req.OperationType,
		OperationID: req.OperationID, ExpiresAt: pgconv.TimeToTimestamptz(req.ExpiresAt)})
}

func (r *Repository) ReservationByOperation(ctx context.Context, organizationID uuid.UUID, operationType, operationID string) (sqlc.WalletReservation, error) {
	return r.queries.GetWalletReservationByOperation(ctx, sqlc.GetWalletReservationByOperationParams{
		OrganizationID: organizationID, OperationType: operationType, OperationID: operationID})
}

func (r *Repository) GetReservation(ctx context.Context, organizationID, id uuid.UUID) (sqlc.WalletReservation, error) {
	return r.queries.GetWalletReservation(ctx, sqlc.GetWalletReservationParams{OrganizationID: organizationID, ID: id})
}

func (r *Repository) LockReservation(ctx context.Context, organizationID, id uuid.UUID) (sqlc.WalletReservation, error) {
	return r.queries.LockWalletReservation(ctx, sqlc.LockWalletReservationParams{OrganizationID: organizationID, ID: id})
}

func (r *Repository) ExtendReservation(ctx context.Context, id uuid.UUID, amount int64, expiresAt time.Time) (sqlc.WalletReservation, error) {
	return r.queries.ExtendWalletReservation(ctx, sqlc.ExtendWalletReservationParams{ID: id, AmountMinor: amount, ExpiresAt: pgconv.TimeToTimestamptz(expiresAt)})
}

func (r *Repository) ReleaseReservation(ctx context.Context, id uuid.UUID, status string) (sqlc.WalletReservation, error) {
	return r.queries.ReleaseWalletReservation(ctx, sqlc.ReleaseWalletReservationParams{ID: id, Status: status})
}

func (r *Repository) CaptureReservation(ctx context.Context, id uuid.UUID, amount int64, transactionID uuid.UUID) (sqlc.WalletReservation, error) {
	return r.queries.CaptureWalletReservation(ctx, sqlc.CaptureWalletReservationParams{ID: id,
		CapturedAmountMinor: &amount, CapturedTransactionID: &transactionID})
}

func (r *Repository) ExpiredReservations(ctx context.Context, limit int32) ([]sqlc.WalletReservation, error) {
	return r.queries.ListExpiredWalletReservations(ctx, limit)
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

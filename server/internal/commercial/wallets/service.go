// Package wallets owns internal prepaid balance mutations. Never expose its
// credit operation to arbitrary customer requests: credits require verified
// payment-provider settlement or an authorized financial adjustment.
package wallets

import (
	"context"
	"errors"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
}

const (
	defaultTransactionLimit int32 = 50
	maxTransactionLimit     int32 = 200
)

func NewService(db *pgxpool.Pool) *Service {
	return &Service{repo: NewRepository(db)}
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !validCurrency(currency) {
		return sqlc.Wallet{}, apperror.NewBadRequest("valid organization and currency are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Wallet{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}

	wallet, err := s.repo.Create(ctx, organizationID, currency)
	if errors.Is(err, pgx.ErrNoRows) {
		wallet, err = s.repo.Get(ctx, organizationID, currency)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound("organization or wallet not found")
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal("create prepaid wallet", err)
	}
	return wallet, nil
}

func (s *Service) Get(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !validCurrency(currency) {
		return sqlc.Wallet{}, apperror.NewBadRequest("valid organization and currency are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Wallet{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}

	wallet, err := s.repo.Get(ctx, organizationID, currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal("get prepaid wallet", err)
	}
	return wallet, nil
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.Wallet, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization is required")
	}
	if s == nil || !s.repo.Available() {
		return nil, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	rows, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list prepaid wallets", err)
	}
	return rows, nil
}

func (s *Service) GetByID(ctx context.Context, organizationID, walletID uuid.UUID) (sqlc.Wallet, error) {
	if organizationID == uuid.Nil || walletID == uuid.Nil {
		return sqlc.Wallet{}, apperror.NewBadRequest("organization and wallet are required")
	}
	if s == nil || !s.repo.Available() {
		return sqlc.Wallet{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	row, err := s.repo.GetByID(ctx, organizationID, walletID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal("get prepaid wallet", err)
	}
	return row, nil
}

func (s *Service) ListTransactions(ctx context.Context, organizationID, walletID uuid.UUID, limit int32) ([]sqlc.WalletTransaction, error) {
	if organizationID == uuid.Nil || walletID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization and wallet are required")
	}
	if limit == 0 {
		limit = defaultTransactionLimit
	}
	if limit < 0 || limit > maxTransactionLimit {
		return nil, apperror.NewBadRequest("transaction limit must be between 1 and 200")
	}
	if s == nil || !s.repo.Available() {
		return nil, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	if _, err := s.repo.GetByID(ctx, organizationID, walletID); errors.Is(err, pgx.ErrNoRows) {
		return nil, apperror.NewNotFound("wallet not found")
	} else if err != nil {
		return nil, apperror.NewInternal("get prepaid wallet", err)
	}
	rows, err := s.repo.ListTransactions(ctx, organizationID, walletID, limit)
	if err != nil {
		return nil, apperror.NewInternal("list wallet transactions", err)
	}
	return rows, nil
}

// Post changes a balance only together with one immutable financial entry.
// The caller must supply a durable, authenticated business reference; it must
// not use a freshly generated reference for retries of the same operation.
// Top-up credits must follow independently verified payment settlement.
func (s *Service) Post(ctx context.Context, entry Entry) (sqlc.WalletTransaction, error) {
	entry.Currency = strings.ToUpper(strings.TrimSpace(entry.Currency))
	if err := validateEntry(entry); err != nil {
		return sqlc.WalletTransaction{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.WalletTransaction{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}

	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("begin wallet transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	repo := s.repo.WithTx(tx)
	wallet, err := repo.Lock(ctx, entry.OrganizationID, entry.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletTransaction{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("lock prepaid wallet", err)
	}

	existing, err := repo.FindTransaction(ctx, wallet.ID, entry.ReferenceType, entry.ReferenceID)
	if err == nil {
		if existing.Direction != entry.Direction ||
			existing.Reason != entry.Reason ||
			existing.AmountMinor != entry.AmountMinor {
			return sqlc.WalletTransaction{}, apperror.NewConflict("wallet reference already used for a different operation")
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletTransaction{}, apperror.NewInternal("read wallet reference", err)
	}

	next, err := nextBalance(wallet.BalanceMinor, entry.Direction, entry.AmountMinor)
	if err != nil {
		return sqlc.WalletTransaction{}, err
	}

	_, err = repo.SetBalance(ctx, wallet, entry.OrganizationID, next)
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("update prepaid balance", err)
	}
	posted, err := repo.Record(ctx, wallet.ID, entry, next)
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("record wallet transaction", err)
	}
	if err := tx.Commit(ctx); err != nil {
		// An uncertain COMMIT outcome must be reconciled using the SAME
		// reference, never by issuing a new business reference.
		return sqlc.WalletTransaction{}, apperror.NewInternal("commit prepaid transaction", err)
	}
	return posted, nil
}

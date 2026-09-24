// Package wallets owns internal prepaid balance mutations. Never expose its
// credit operation to arbitrary customer requests: credits require verified
// payment-provider settlement or an authorized financial adjustment.
package wallets

import (
	"context"
	"errors"
	"math"
	"regexp"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

var referenceLabel = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

type Service struct {
	db *pgxpool.Pool
}

func NewService(db *pgxpool.Pool) *Service {
	return &Service{db: db}
}

type Entry struct {
	OrganizationID uuid.UUID
	Currency       string
	Direction      string
	Reason         string
	AmountMinor    int64
	ReferenceType  string
	ReferenceID    uuid.UUID
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, currency string) (sqlc.Wallet, error) {
	currency = strings.ToUpper(strings.TrimSpace(currency))
	if organizationID == uuid.Nil || !validCurrency(currency) {
		return sqlc.Wallet{}, apperror.NewBadRequest("valid organization and currency are required")
	}
	if s == nil || s.db == nil {
		return sqlc.Wallet{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	q := sqlc.New(s.db)
	wallet, err := q.CreatePrepaidWallet(ctx, sqlc.CreatePrepaidWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		wallet, err = q.GetPrepaidWallet(ctx, sqlc.GetPrepaidWalletParams{
			OrganizationID: organizationID,
			Currency:       currency,
		})
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
	if s == nil || s.db == nil {
		return sqlc.Wallet{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	wallet, err := sqlc.New(s.db).GetPrepaidWallet(ctx, sqlc.GetPrepaidWalletParams{
		OrganizationID: organizationID,
		Currency:       currency,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal("get prepaid wallet", err)
	}
	return wallet, nil
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
	if s == nil || s.db == nil {
		return sqlc.WalletTransaction{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("begin wallet transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	q := sqlc.New(tx)
	wallet, err := q.LockPrepaidWallet(ctx, sqlc.LockPrepaidWalletParams{
		OrganizationID: entry.OrganizationID,
		Currency:       entry.Currency,
	})
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletTransaction{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("lock prepaid wallet", err)
	}

	existing, err := q.GetWalletTransactionByReference(ctx, sqlc.GetWalletTransactionByReferenceParams{
		WalletID:      wallet.ID,
		ReferenceType: entry.ReferenceType,
		ReferenceID:   entry.ReferenceID,
	})
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

	_, err = q.SetPrepaidWalletBalance(ctx, sqlc.SetPrepaidWalletBalanceParams{
		WalletID:             wallet.ID,
		OrganizationID:       entry.OrganizationID,
		PreviousBalanceMinor: wallet.BalanceMinor,
		BalanceMinor:         next,
	})
	if err != nil {
		return sqlc.WalletTransaction{}, apperror.NewInternal("update prepaid balance", err)
	}

	posted, err := q.CreateWalletTransaction(ctx, sqlc.CreateWalletTransactionParams{
		WalletID:          wallet.ID,
		Direction:         entry.Direction,
		Reason:            entry.Reason,
		AmountMinor:       entry.AmountMinor,
		BalanceAfterMinor: next,
		ReferenceType:     entry.ReferenceType,
		ReferenceID:       entry.ReferenceID,
	})
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

func validateEntry(entry Entry) error {
	if entry.OrganizationID == uuid.Nil || !validCurrency(entry.Currency) {
		return apperror.NewBadRequest("valid organization and currency are required")
	}
	if entry.Direction != "credit" && entry.Direction != "debit" {
		return apperror.NewBadRequest("wallet direction must be credit or debit")
	}
	if entry.AmountMinor <= 0 {
		return apperror.NewBadRequest("wallet amount must be positive")
	}
	if !referenceLabel.MatchString(entry.Reason) ||
		!referenceLabel.MatchString(entry.ReferenceType) ||
		entry.ReferenceID == uuid.Nil {
		return apperror.NewBadRequest("valid wallet reason and durable reference are required")
	}
	return nil
}

func validCurrency(currency string) bool {
	if len(currency) != 3 {
		return false
	}
	for _, letter := range currency {
		if letter < 'A' || letter > 'Z' {
			return false
		}
	}
	return true
}

func nextBalance(current int64, direction string, amount int64) (int64, error) {
	if current < 0 || amount <= 0 {
		return 0, apperror.NewBadRequest("invalid prepaid balance or amount")
	}
	switch direction {
	case "debit":
		if amount > current {
			return 0, apperror.NewPaymentRequired("insufficient prepaid balance")
		}
		return current - amount, nil
	case "credit":
		if amount > math.MaxInt64-current {
			return 0, apperror.NewConflict("prepaid balance limit exceeded")
		}
		return current + amount, nil
	default:
		return 0, apperror.NewBadRequest("invalid prepaid transaction direction")
	}
}

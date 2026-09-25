// Package wallets owns internal prepaid balance mutations. Never expose its
// credit operation to arbitrary customer requests: credits require verified
// payment-provider settlement or an authorized financial adjustment.
package wallets

import (
	"context"
	"errors"
	"strings"
	"time"

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

func (s *Service) Reserve(ctx context.Context, req ReserveRequest) (sqlc.WalletReservation, error) {
	if err := validateReserve(&req); err != nil {
		return sqlc.WalletReservation{}, err
	}
	if s == nil || !s.repo.Available() {
		return sqlc.WalletReservation{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("begin wallet reservation", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo := s.repo.WithTx(tx)
	wallet, err := repo.Lock(ctx, req.OrganizationID, req.Currency)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletReservation{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("lock reservation wallet", err)
	}
	existing, err := repo.ReservationByOperation(ctx, req.OrganizationID, req.OperationType, req.OperationID)
	if err == nil {
		if existing.WalletID != wallet.ID || existing.AmountMinor != req.AmountMinor || !existing.ExpiresAt.Time.Equal(req.ExpiresAt) {
			return sqlc.WalletReservation{}, apperror.NewConflict("reservation operation was used for different terms")
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletReservation{}, apperror.NewInternal("read wallet reservation", err)
	}
	if req.AmountMinor > wallet.BalanceMinor-wallet.ReservedMinor {
		return sqlc.WalletReservation{}, apperror.NewPaymentRequired("insufficient available prepaid balance")
	}
	reservation, err := repo.CreateReservation(ctx, wallet.ID, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletReservation{}, apperror.NewConflict("reservation operation already exists")
	}
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("create wallet reservation", err)
	}
	if _, err = repo.SetAmounts(ctx, wallet, req.OrganizationID, wallet.BalanceMinor, wallet.ReservedMinor+req.AmountMinor); err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("reserve prepaid funds", err)
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("commit wallet reservation", err)
	}
	return reservation, nil
}

func (s *Service) Extend(ctx context.Context, req ExtendRequest) (sqlc.WalletReservation, error) {
	if err := validateExtend(&req); err != nil {
		return sqlc.WalletReservation{}, err
	}
	return s.mutateReservation(ctx, req.OrganizationID, req.ReservationID, func(repo *Repository, wallet sqlc.Wallet, reservation sqlc.WalletReservation) (sqlc.WalletReservation, error) {
		if reservation.AmountMinor == req.AmountMinor && reservation.ExpiresAt.Time.Equal(req.ExpiresAt) {
			return reservation, nil
		}
		if reservation.Status != "active" || !reservation.ExpiresAt.Time.After(time.Now()) {
			return sqlc.WalletReservation{}, apperror.NewConflict("reservation is not active")
		}
		if req.AmountMinor <= reservation.AmountMinor {
			return sqlc.WalletReservation{}, apperror.NewConflict("extension must increase reservation amount")
		}
		delta := req.AmountMinor - reservation.AmountMinor
		if delta > wallet.BalanceMinor-wallet.ReservedMinor {
			return sqlc.WalletReservation{}, apperror.NewPaymentRequired("insufficient available prepaid balance")
		}
		if !req.ExpiresAt.After(reservation.ExpiresAt.Time) {
			return sqlc.WalletReservation{}, apperror.NewBadRequest("extension must increase reservation expiration")
		}
		row, err := repo.ExtendReservation(ctx, reservation.ID, req.AmountMinor, req.ExpiresAt)
		if err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("extend wallet reservation", err)
		}
		if _, err = repo.SetAmounts(ctx, wallet, req.OrganizationID, wallet.BalanceMinor, wallet.ReservedMinor+delta); err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("extend reserved funds", err)
		}
		return row, nil
	})
}

func (s *Service) Release(ctx context.Context, organizationID, reservationID uuid.UUID) (sqlc.WalletReservation, error) {
	return s.release(ctx, organizationID, reservationID, "released", false)
}

func (s *Service) Expire(ctx context.Context, organizationID, reservationID uuid.UUID) (sqlc.WalletReservation, error) {
	return s.release(ctx, organizationID, reservationID, "expired", false)
}

func (s *Service) release(ctx context.Context, organizationID, reservationID uuid.UUID, status string, verifiedManagedSettlement bool) (sqlc.WalletReservation, error) {
	if organizationID == uuid.Nil || reservationID == uuid.Nil {
		return sqlc.WalletReservation{}, apperror.NewBadRequest("organization and reservation are required")
	}
	return s.mutateReservation(ctx, organizationID, reservationID, func(repo *Repository, wallet sqlc.Wallet, reservation sqlc.WalletReservation) (sqlc.WalletReservation, error) {
		// A managed call can remain billable after its reservation expiry.
		// Only verified settlement may release these funds, never a timer.
		if reservation.OperationType == "managed_call" &&
			(status == "expired" || !verifiedManagedSettlement) {
			return sqlc.WalletReservation{}, apperror.NewConflict(
				"managed call authorization requires verified settlement",
			)
		}
		if reservation.Status == status {
			return reservation, nil
		}
		if reservation.Status != "active" {
			return sqlc.WalletReservation{}, apperror.NewConflict("reservation is not active")
		}
		if status == "expired" && reservation.ExpiresAt.Time.After(time.Now()) {
			return sqlc.WalletReservation{}, apperror.NewConflict("reservation has not expired")
		}
		row, err := repo.ReleaseReservation(ctx, reservation.ID, status)
		if err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("release wallet reservation", err)
		}
		if _, err = repo.SetAmounts(ctx, wallet, organizationID, wallet.BalanceMinor, wallet.ReservedMinor-reservation.AmountMinor); err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("release reserved funds", err)
		}
		return row, nil
	})
}

func (s *Service) Capture(ctx context.Context, req CaptureRequest) (ReservationResult, error) {
	if err := validateCapture(req); err != nil {
		return ReservationResult{}, err
	}
	var transaction sqlc.WalletTransaction
	reservation, err := s.mutateReservation(ctx, req.OrganizationID, req.ReservationID, func(repo *Repository, wallet sqlc.Wallet, reservation sqlc.WalletReservation) (sqlc.WalletReservation, error) {
		if reservation.Status == "captured" && reservation.CapturedTransactionID != nil {
			entry, getErr := repo.Transaction(ctx, *reservation.CapturedTransactionID)
			if getErr != nil {
				return sqlc.WalletReservation{}, apperror.NewInternal("read reservation capture", getErr)
			}
			if reservation.CapturedAmountMinor == nil || *reservation.CapturedAmountMinor != req.AmountMinor || entry.ReferenceType != req.ReferenceType || entry.ReferenceID != req.ReferenceID {
				return sqlc.WalletReservation{}, apperror.NewConflict("reservation was captured with different terms")
			}
			transaction = entry
			return reservation, nil
		}
		if reservation.Status != "active" || req.AmountMinor > reservation.AmountMinor {
			return sqlc.WalletReservation{}, apperror.NewConflict("reservation cannot cover capture")
		}
		next := wallet.BalanceMinor - req.AmountMinor
		if _, err := repo.SetAmounts(ctx, wallet, req.OrganizationID, next, wallet.ReservedMinor-reservation.AmountMinor); err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("capture reserved funds", err)
		}
		entry, err := repo.Record(ctx, wallet.ID, Entry{OrganizationID: req.OrganizationID, Currency: wallet.Currency,
			Direction: "debit", Reason: req.Reason, AmountMinor: req.AmountMinor, ReferenceType: req.ReferenceType, ReferenceID: req.ReferenceID}, next)
		if err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("record reservation capture", err)
		}
		row, err := repo.CaptureReservation(ctx, reservation.ID, req.AmountMinor, entry.ID)
		if err != nil {
			return sqlc.WalletReservation{}, apperror.NewInternal("complete reservation capture", err)
		}
		transaction = entry
		return row, nil
	})
	if err != nil {
		return ReservationResult{}, err
	}
	return ReservationResult{Reservation: reservation, Transaction: &transaction}, nil
}

type reservationMutation func(*Repository, sqlc.Wallet, sqlc.WalletReservation) (sqlc.WalletReservation, error)

func (s *Service) mutateReservation(ctx context.Context, organizationID, reservationID uuid.UUID, mutate reservationMutation) (sqlc.WalletReservation, error) {
	if s == nil || !s.repo.Available() {
		return sqlc.WalletReservation{}, apperror.NewServiceUnavailable("prepaid wallets are not configured", nil)
	}
	current, err := s.repo.GetReservation(ctx, organizationID, reservationID)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletReservation{}, apperror.NewNotFound("reservation not found")
	}
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("get wallet reservation", err)
	}
	tx, err := s.repo.Begin(ctx)
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("begin reservation transaction", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()
	repo := s.repo.WithTx(tx)
	wallet, err := repo.LockByID(ctx, organizationID, current.WalletID)
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("lock reservation wallet", err)
	}
	reservation, err := repo.LockReservation(ctx, organizationID, reservationID)
	if err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("lock wallet reservation", err)
	}
	result, err := mutate(repo, wallet, reservation)
	if err != nil {
		return sqlc.WalletReservation{}, err
	}
	if err := tx.Commit(ctx); err != nil {
		return sqlc.WalletReservation{}, apperror.NewInternal("commit reservation transaction", err)
	}
	return result, nil
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
	if entry.Direction == "debit" && next < wallet.ReservedMinor {
		return sqlc.WalletTransaction{}, apperror.NewPaymentRequired("insufficient available prepaid balance")
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

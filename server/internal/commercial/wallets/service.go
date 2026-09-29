package wallets

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Service struct {
	repo *Repository
	db   *pgxpool.Pool
	now  func() time.Time
}

func NewService(
	repo *Repository,
	db *pgxpool.Pool,
) *Service {
	return &Service{
		repo: repo,
		db:   db,
		now:  time.Now,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (Wallet, error) {
	if err := validateCreateRequest(req); err != nil {
		return Wallet{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Create(
		ctx,
		req.OrganizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = s.repo.GetActive(
			ctx,
			req.OrganizationID,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return Wallet{}, apperror.NewInternal("create wallet", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
) (Wallet, error) {
	if organizationID == uuid.Nil {
		return Wallet{}, apperror.NewNotFound("wallet not found")
	}

	row, err := s.repo.GetActive(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return Wallet{}, apperror.NewInternal("get wallet", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Credit(
	ctx context.Context,
	req MovementRequest,
) (LedgerEntry, error) {
	return s.applyMovement(
		ctx,
		DirectionCredit,
		req,
	)
}

func (s *Service) Debit(
	ctx context.Context,
	req MovementRequest,
) (LedgerEntry, error) {
	return s.applyMovement(
		ctx,
		DirectionDebit,
		req,
	)
}

func (s *Service) Hold(
	ctx context.Context,
	req HoldRequest,
) (Hold, error) {
	if err := normalizeHoldRequest(&req); err != nil {
		return Hold{}, apperror.NewBadRequest(err.Error())
	}

	existing, err := s.repo.GetHoldByOperation(
		ctx,
		req.OrganizationID,
		req.OperationID,
	)
	if err == nil {
		if !sameHold(existing, req) {
			return Hold{}, apperror.NewConflict("wallet hold conflicts with existing operation")
		}

		return holdFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewInternal("read wallet hold", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Hold{}, apperror.NewInternal("begin wallet hold", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := s.repo.WithTx(tx)

	wallet, err := repo.LockActive(
		ctx,
		req.OrganizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return Hold{}, apperror.NewInternal("lock wallet", err)
	}

	existing, err = repo.GetHoldByOperation(
		ctx,
		req.OrganizationID,
		req.OperationID,
	)
	if err == nil {
		if !sameHold(existing, req) {
			return Hold{}, apperror.NewConflict("wallet hold conflicts with existing operation")
		}
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, apperror.NewInternal("commit wallet hold replay", err)
		}

		return holdFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewInternal("read locked wallet hold", err)
	}

	if wallet.BalanceMicros-wallet.ReservedMicros < req.AmountMicros {
		return Hold{}, apperror.NewPaymentRequired("insufficient wallet balance")
	}

	if _, err := repo.ReserveBalance(
		ctx,
		req.OrganizationID,
		req.AmountMicros,
	); errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewPaymentRequired("insufficient wallet balance")
	} else if err != nil {
		return Hold{}, apperror.NewInternal("reserve wallet balance", err)
	}

	hold, err := repo.CreateHold(
		ctx,
		sqlc.CreateWalletHoldParams{
			WalletID:       wallet.ID,
			OrganizationID: req.OrganizationID,
			OperationID:    req.OperationID,
			AmountMicros:   req.AmountMicros,
			Reason:         req.Reason,
			ReferenceType:  req.ReferenceType,
			ReferenceID:    req.ReferenceID,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return Hold{}, apperror.NewConflict("wallet hold conflicts with existing operation")
		}

		return Hold{}, apperror.NewInternal("create wallet hold", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Hold{}, apperror.NewInternal("commit wallet hold", err)
	}

	return holdFromRow(hold), nil
}

func (s *Service) GetHold(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (Hold, error) {
	if err := validateHoldOperation(organizationID, operationID); err != nil {
		return Hold{}, apperror.NewBadRequest(err.Error())
	}

	hold, err := s.repo.GetHoldByOperation(
		ctx,
		organizationID,
		operationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewNotFound("wallet hold not found")
	}
	if err != nil {
		return Hold{}, apperror.NewInternal("get wallet hold", err)
	}

	return holdFromRow(hold), nil
}

func (s *Service) Capture(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (LedgerEntry, error) {
	return s.capture(
		ctx,
		organizationID,
		operationID,
		0,
	)
}

// CaptureAmount settles a hold for an amount up to the authorized maximum and
// releases the unused reservation in the same transaction.
func (s *Service) CaptureAmount(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
	amountMicros int64,
) (LedgerEntry, error) {
	if amountMicros <= 0 {
		return LedgerEntry{}, apperror.NewBadRequest("capture amount must be positive")
	}

	return s.capture(
		ctx,
		organizationID,
		operationID,
		amountMicros,
	)
}

func (s *Service) capture(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
	amountMicros int64,
) (LedgerEntry, error) {
	if err := validateHoldOperation(organizationID, operationID); err != nil {
		return LedgerEntry{}, apperror.NewBadRequest(err.Error())
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("begin wallet hold capture", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := s.repo.WithTx(tx)

	if _, err := repo.LockForSettlement(
		ctx,
		organizationID,
	); errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewNotFound("wallet not found")
	} else if err != nil {
		return LedgerEntry{}, apperror.NewInternal("lock wallet for hold capture", err)
	}

	hold, err := repo.GetHoldByOperation(
		ctx,
		organizationID,
		operationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewNotFound("wallet hold not found")
	}
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("get wallet hold", err)
	}

	switch hold.Status {
	case HoldStatusCaptured:
		entry, err := repo.GetLedgerEntryByOperation(
			ctx,
			organizationID,
			operationID,
		)
		if errors.Is(err, pgx.ErrNoRows) {
			return LedgerEntry{}, apperror.NewInternal(
				"captured wallet hold is missing ledger entry",
				err,
			)
		}
		if err != nil {
			return LedgerEntry{}, apperror.NewInternal("get captured wallet ledger entry", err)
		}
		if amountMicros > 0 && entry.AmountMicros != amountMicros {
			return LedgerEntry{}, apperror.NewConflict("wallet capture amount conflicts with existing ledger entry")
		}
		if err := tx.Commit(ctx); err != nil {
			return LedgerEntry{}, apperror.NewInternal("commit wallet hold capture replay", err)
		}

		return ledgerEntryFromRow(entry), nil
	case HoldStatusReleased:
		return LedgerEntry{}, apperror.NewConflict("released wallet hold cannot be captured")
	case HoldStatusActive:
	default:
		return LedgerEntry{}, apperror.NewConflict("wallet hold state does not allow capture")
	}
	if amountMicros == 0 {
		amountMicros = hold.AmountMicros
	}
	if amountMicros > hold.AmountMicros {
		return LedgerEntry{}, apperror.NewPaymentRequired("capture amount exceeds wallet authorization")
	}

	if _, err := repo.ReleaseReservedBalance(
		ctx,
		organizationID,
		hold.AmountMicros,
	); errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewConflict("wallet state does not allow hold capture")
	} else if err != nil {
		return LedgerEntry{}, apperror.NewInternal("release captured wallet reservation", err)
	}

	updated, err := repo.ApplyBalance(
		ctx,
		organizationID,
		-amountMicros,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewConflict("wallet state does not allow hold capture")
	}
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("capture wallet balance", err)
	}

	entry, err := repo.CreateLedgerEntry(
		ctx,
		sqlc.CreateWalletLedgerEntryParams{
			OperationID:        hold.OperationID,
			Direction:          DirectionDebit,
			Reason:             hold.Reason,
			AmountMicros:       amountMicros,
			BalanceAfterMicros: updated.BalanceMicros,
			ReferenceType:      hold.ReferenceType,
			ReferenceID:        hold.ReferenceID,
			OccurredAt:         pgconv.TimeToTimestamptz(s.now().UTC()),
			OrganizationID:     hold.OrganizationID,
		},
	)
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("create wallet hold ledger entry", err)
	}

	if _, err := repo.MarkHoldCaptured(
		ctx,
		organizationID,
		operationID,
	); errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewConflict("wallet hold state does not allow capture")
	} else if err != nil {
		return LedgerEntry{}, apperror.NewInternal("mark wallet hold captured", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return LedgerEntry{}, apperror.NewInternal("commit wallet hold capture", err)
	}

	return ledgerEntryFromRow(entry), nil
}

func (s *Service) Release(
	ctx context.Context,
	organizationID uuid.UUID,
	operationID uuid.UUID,
) (Hold, error) {
	if err := validateHoldOperation(organizationID, operationID); err != nil {
		return Hold{}, apperror.NewBadRequest(err.Error())
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return Hold{}, apperror.NewInternal("begin wallet hold release", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := s.repo.WithTx(tx)

	if _, err := repo.LockForSettlement(
		ctx,
		organizationID,
	); errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewNotFound("wallet not found")
	} else if err != nil {
		return Hold{}, apperror.NewInternal("lock wallet for hold release", err)
	}

	hold, err := repo.GetHoldByOperation(
		ctx,
		organizationID,
		operationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewNotFound("wallet hold not found")
	}
	if err != nil {
		return Hold{}, apperror.NewInternal("get wallet hold", err)
	}

	switch hold.Status {
	case HoldStatusReleased:
		if err := tx.Commit(ctx); err != nil {
			return Hold{}, apperror.NewInternal("commit wallet hold release replay", err)
		}

		return holdFromRow(hold), nil
	case HoldStatusCaptured:
		return Hold{}, apperror.NewConflict("captured wallet hold cannot be released")
	case HoldStatusActive:
	default:
		return Hold{}, apperror.NewConflict("wallet hold state does not allow release")
	}

	if _, err := repo.ReleaseReservedBalance(
		ctx,
		organizationID,
		hold.AmountMicros,
	); errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewConflict("wallet state does not allow hold release")
	} else if err != nil {
		return Hold{}, apperror.NewInternal("release wallet balance", err)
	}

	hold, err = repo.MarkHoldReleased(
		ctx,
		organizationID,
		operationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Hold{}, apperror.NewConflict("wallet hold state does not allow release")
	}
	if err != nil {
		return Hold{}, apperror.NewInternal("mark wallet hold released", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return Hold{}, apperror.NewInternal("commit wallet hold release", err)
	}

	return holdFromRow(hold), nil
}

func (s *Service) ListLedgerEntries(
	ctx context.Context,
	req ListLedgerRequest,
) ([]LedgerEntry, error) {
	if err := normalizeListLedgerRequest(&req); err != nil {
		return nil, apperror.NewBadRequest(err.Error())
	}

	rows, err := s.repo.ListLedgerEntries(
		ctx,
		req,
	)
	if err != nil {
		return nil, apperror.NewInternal("list wallet ledger entries", err)
	}

	result := make([]LedgerEntry, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			ledgerEntryFromRow(row),
		)
	}

	return result, nil
}

func (s *Service) Freeze(
	ctx context.Context,
	organizationID uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Freeze(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperror.NewConflict("wallet state does not allow operation")
	}
	if err != nil {
		return Wallet{}, apperror.NewInternal("freeze wallet", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Activate(
	ctx context.Context,
	organizationID uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Activate(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperror.NewConflict("wallet state does not allow operation")
	}
	if err != nil {
		return Wallet{}, apperror.NewInternal("activate wallet", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Close(
	ctx context.Context,
	organizationID uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Close(
		ctx,
		organizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, apperror.NewConflict("wallet state does not allow operation")
	}
	if err != nil {
		return Wallet{}, apperror.NewInternal("close wallet", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) applyMovement(
	ctx context.Context,
	direction string,
	req MovementRequest,
) (LedgerEntry, error) {
	if err := normalizeMovementRequest(&req); err != nil {
		return LedgerEntry{}, apperror.NewBadRequest(err.Error())
	}

	existing, err := s.repo.GetLedgerEntryByOperation(
		ctx,
		req.OrganizationID,
		req.OperationID,
	)
	if err == nil {
		if !sameMovement(existing, direction, req) {
			return LedgerEntry{}, apperror.NewConflict("wallet operation conflicts with existing ledger entry")
		}

		return ledgerEntryFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewInternal("read wallet operation", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("begin wallet movement", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := s.repo.WithTx(tx)

	wallet, err := repo.LockActive(
		ctx,
		req.OrganizationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewNotFound("wallet not found")
	}
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("lock wallet", err)
	}

	existing, err = repo.GetLedgerEntryByOperation(
		ctx,
		req.OrganizationID,
		req.OperationID,
	)
	if err == nil {
		if !sameMovement(existing, direction, req) {
			return LedgerEntry{}, apperror.NewConflict("wallet operation conflicts with existing ledger entry")
		}
		if err := tx.Commit(ctx); err != nil {
			return LedgerEntry{}, apperror.NewInternal("commit wallet replay", err)
		}

		return ledgerEntryFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, apperror.NewInternal("read locked wallet operation", err)
	}

	delta := req.AmountMicros
	if direction == DirectionDebit {
		if wallet.BalanceMicros-wallet.ReservedMicros < req.AmountMicros {
			return LedgerEntry{}, apperror.NewPaymentRequired("insufficient wallet balance")
		}
		delta = -req.AmountMicros
	}

	updated, err := repo.ApplyBalance(
		ctx,
		req.OrganizationID,
		delta,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if direction == DirectionDebit {
			return LedgerEntry{}, apperror.NewPaymentRequired("insufficient wallet balance")
		}
		return LedgerEntry{}, apperror.NewConflict("wallet state does not allow operation")
	}
	if err != nil {
		return LedgerEntry{}, apperror.NewInternal("apply wallet balance", err)
	}

	occurredAt := req.OccurredAt
	if occurredAt.IsZero() {
		occurredAt = s.now().UTC()
	}

	entry, err := repo.CreateLedgerEntry(
		ctx,
		sqlc.CreateWalletLedgerEntryParams{
			OperationID:        req.OperationID,
			Direction:          direction,
			Reason:             req.Reason,
			AmountMicros:       req.AmountMicros,
			BalanceAfterMicros: updated.BalanceMicros,
			ReferenceType:      req.ReferenceType,
			ReferenceID:        req.ReferenceID,
			OccurredAt:         pgconv.TimeToTimestamptz(occurredAt),
			OrganizationID:     req.OrganizationID,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return LedgerEntry{}, apperror.NewConflict("wallet operation conflicts with existing ledger entry")
		}

		return LedgerEntry{}, apperror.NewInternal("create wallet ledger entry", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return LedgerEntry{}, apperror.NewInternal("commit wallet movement", err)
	}

	return ledgerEntryFromRow(entry), nil
}

func sameMovement(
	existing sqlc.WalletLedgerEntry,
	direction string,
	req MovementRequest,
) bool {
	return existing.Direction == direction &&
		existing.Reason == req.Reason &&
		existing.AmountMicros == req.AmountMicros &&
		equalOptionalString(existing.ReferenceType, req.ReferenceType) &&
		equalOptionalUUID(existing.ReferenceID, req.ReferenceID)
}

func sameHold(
	existing sqlc.WalletHold,
	req HoldRequest,
) bool {
	return existing.AmountMicros == req.AmountMicros &&
		existing.Reason == req.Reason &&
		equalOptionalString(existing.ReferenceType, req.ReferenceType) &&
		equalOptionalUUID(existing.ReferenceID, req.ReferenceID)
}

func equalOptionalString(
	left *string,
	right *string,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalOptionalUUID(
	left *uuid.UUID,
	right *uuid.UUID,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func walletFromRow(row sqlc.Wallet) Wallet {
	return Wallet{
		ID:              row.ID,
		OrganizationID:  row.OrganizationID,
		Currency:        row.Currency,
		Status:          row.Status,
		BalanceMicros:   row.BalanceMicros,
		ReservedMicros:  row.ReservedMicros,
		AvailableMicros: row.BalanceMicros - row.ReservedMicros,
		CreatedAt:       pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:       pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func holdFromRow(row sqlc.WalletHold) Hold {
	return Hold{
		ID:             row.ID,
		WalletID:       row.WalletID,
		OrganizationID: row.OrganizationID,
		OperationID:    row.OperationID,
		AmountMicros:   row.AmountMicros,
		Reason:         row.Reason,
		ReferenceType:  row.ReferenceType,
		ReferenceID:    row.ReferenceID,
		Status:         row.Status,
		CapturedAt:     pgconv.TimestamptzToTimePtr(row.CapturedAt),
		ReleasedAt:     pgconv.TimestamptzToTimePtr(row.ReleasedAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func ledgerEntryFromRow(row sqlc.WalletLedgerEntry) LedgerEntry {
	return LedgerEntry{
		ID:                 row.ID,
		WalletID:           row.WalletID,
		OrganizationID:     row.OrganizationID,
		OperationID:        row.OperationID,
		Direction:          row.Direction,
		Reason:             row.Reason,
		AmountMicros:       row.AmountMicros,
		BalanceAfterMicros: row.BalanceAfterMicros,
		ReferenceType:      row.ReferenceType,
		ReferenceID:        row.ReferenceID,
		OccurredAt:         pgconv.TimestamptzToTime(row.OccurredAt),
		CreatedAt:          pgconv.TimestamptzToTime(row.CreatedAt),
	}
}

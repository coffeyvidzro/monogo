package wallets

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
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
	if err := normalizeCreateRequest(&req); err != nil {
		return Wallet{}, err
	}

	row, err := s.repo.Create(
		ctx,
		req.OrganizationID,
		req.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		row, err = s.repo.GetActiveByCurrency(
			ctx,
			req.OrganizationID,
			req.Currency,
		)
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("create wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Wallet, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Wallet{}, ErrNotFound
	}

	row, err := s.repo.GetActive(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("get wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) GetByCurrency(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (Wallet, error) {
	req := CreateRequest{
		OrganizationID: organizationID,
		Currency:       currency,
	}
	if err := normalizeCreateRequest(&req); err != nil {
		return Wallet{}, err
	}

	row, err := s.repo.GetActiveByCurrency(
		ctx,
		req.OrganizationID,
		req.Currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrNotFound
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("get wallet by currency: %w", err)
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

func (s *Service) ListLedgerEntries(
	ctx context.Context,
	req ListLedgerRequest,
) ([]LedgerEntry, error) {
	if err := normalizeListLedgerRequest(&req); err != nil {
		return nil, err
	}

	rows, err := s.repo.ListLedgerEntries(
		ctx,
		req,
	)
	if err != nil {
		return nil, fmt.Errorf("list wallet ledger entries: %w", err)
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
	id uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Freeze(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrInvalidState
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("freeze wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Activate(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Activate(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrInvalidState
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("activate wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) Close(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Wallet, error) {
	row, err := s.repo.Close(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Wallet{}, ErrInvalidState
	}
	if err != nil {
		return Wallet{}, fmt.Errorf("close wallet: %w", err)
	}

	return walletFromRow(row), nil
}

func (s *Service) applyMovement(
	ctx context.Context,
	direction string,
	req MovementRequest,
) (LedgerEntry, error) {
	if err := normalizeMovementRequest(&req); err != nil {
		return LedgerEntry{}, err
	}

	existing, err := s.repo.GetLedgerEntryByOperation(
		ctx,
		req.OrganizationID,
		req.WalletID,
		req.OperationID,
	)
	if err == nil {
		if !sameMovement(existing, direction, req) {
			return LedgerEntry{}, ErrOperationConflict
		}

		return ledgerEntryFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, fmt.Errorf("read wallet operation: %w", err)
	}

	tx, err := s.db.Begin(ctx)
	if err != nil {
		return LedgerEntry{}, fmt.Errorf("begin wallet movement: %w", err)
	}
	defer func() {
		_ = tx.Rollback(ctx)
	}()

	repo := s.repo.WithTx(tx)

	wallet, err := repo.LockActive(
		ctx,
		req.OrganizationID,
		req.WalletID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, ErrNotFound
	}
	if err != nil {
		return LedgerEntry{}, fmt.Errorf("lock wallet: %w", err)
	}

	existing, err = repo.GetLedgerEntryByOperation(
		ctx,
		req.OrganizationID,
		req.WalletID,
		req.OperationID,
	)
	if err == nil {
		if !sameMovement(existing, direction, req) {
			return LedgerEntry{}, ErrOperationConflict
		}
		if err := tx.Commit(ctx); err != nil {
			return LedgerEntry{}, fmt.Errorf("commit wallet replay: %w", err)
		}

		return ledgerEntryFromRow(existing), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return LedgerEntry{}, fmt.Errorf("read locked wallet operation: %w", err)
	}

	delta := req.AmountMicros
	if direction == DirectionDebit {
		if wallet.BalanceMicros < req.AmountMicros {
			return LedgerEntry{}, ErrInsufficientBalance
		}
		delta = -req.AmountMicros
	}

	updated, err := repo.ApplyBalance(
		ctx,
		req.OrganizationID,
		req.WalletID,
		delta,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		if direction == DirectionDebit {
			return LedgerEntry{}, ErrInsufficientBalance
		}
		return LedgerEntry{}, ErrInvalidState
	}
	if err != nil {
		return LedgerEntry{}, fmt.Errorf("apply wallet balance: %w", err)
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
			WalletID:           req.WalletID,
			OrganizationID:     req.OrganizationID,
		},
	)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return LedgerEntry{}, ErrOperationConflict
		}

		return LedgerEntry{}, fmt.Errorf("create wallet ledger entry: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return LedgerEntry{}, fmt.Errorf("commit wallet movement: %w", err)
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
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Currency:       row.Currency,
		Status:         row.Status,
		BalanceMicros:  row.BalanceMicros,
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

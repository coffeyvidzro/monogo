package ledger

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultListLimit int32 = 50
	maxListLimit     int32 = 200
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("billing ledger: repository is required")
	}

	return &Service{
		repo: repo,
	}
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	if err := validateEntryIdentity(
		organizationID,
		id,
	); err != nil {
		return sqlc.WalletLedgerEntry{}, err
	}

	entry, err := s.repo.Get(
		ctx,
		organizationID,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletLedgerEntry{}, apperror.NewNotFound(
			"ledger entry not found",
		)
	}
	if err != nil {
		return sqlc.WalletLedgerEntry{}, apperror.NewInternal(
			"get wallet ledger entry",
			err,
		)
	}

	return entry, nil
}

func (s *Service) ListWallet(
	ctx context.Context,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletLedgerEntry, error) {
	if err := validateWalletIdentity(
		organizationID,
		walletID,
	); err != nil {
		return nil, err
	}

	limit, err := normalizeListLimit(limit)
	if err != nil {
		return nil, err
	}

	entries, err := s.repo.ListWallet(
		ctx,
		organizationID,
		walletID,
		limit,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list wallet ledger entries",
			err,
		)
	}

	return entries, nil
}

func (s *Service) ListCharge(
	ctx context.Context,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) ([]sqlc.WalletLedgerEntry, error) {
	if err := validateChargeIdentity(
		organizationID,
		chargeID,
	); err != nil {
		return nil, err
	}

	entries, err := s.repo.ListCharge(
		ctx,
		organizationID,
		chargeID,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list charge ledger entries",
			err,
		)
	}

	return entries, nil
}

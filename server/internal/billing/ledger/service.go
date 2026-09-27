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
	return &Service{repo: repo}
}

func (s *Service) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.WalletLedgerEntry, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return sqlc.WalletLedgerEntry{}, apperror.NewBadRequest(
			"organization and ledger entry are required",
		)
	}
	entry, err := s.repo.Get(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletLedgerEntry{}, apperror.NewNotFound("ledger entry not found")
	}
	if err != nil {
		return sqlc.WalletLedgerEntry{}, apperror.NewInternal("get wallet ledger entry", err)
	}
	return entry, nil
}

func (s *Service) ListWallet(
	ctx context.Context,
	organizationID, walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletLedgerEntry, error) {
	if organizationID == uuid.Nil || walletID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization and wallet are required")
	}
	if limit == 0 {
		limit = defaultListLimit
	}
	if limit < 1 || limit > maxListLimit {
		return nil, apperror.NewBadRequest("ledger limit must be between 1 and 200")
	}
	entries, err := s.repo.ListWallet(ctx, organizationID, walletID, limit)
	if err != nil {
		return nil, apperror.NewInternal("list wallet ledger entries", err)
	}
	return entries, nil
}

func (s *Service) ListCharge(
	ctx context.Context,
	organizationID, chargeID uuid.UUID,
) ([]sqlc.WalletLedgerEntry, error) {
	if organizationID == uuid.Nil || chargeID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization and charge are required")
	}
	entries, err := s.repo.ListCharge(ctx, organizationID, chargeID)
	if err != nil {
		return nil, apperror.NewInternal("list charge ledger entries", err)
	}
	return entries, nil
}

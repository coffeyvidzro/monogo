package wallets

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

const (
	defaultEventLimit int32 = 50
	maxEventLimit     int32 = 200
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("billing wallets: repository is required")
	}

	return &Service{
		repo: repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	if organizationID == uuid.Nil {
		return sqlc.Wallet{}, apperror.NewBadRequest(
			"organization context required",
		)
	}

	currency, err := normalizeCurrency(currency)
	if err != nil {
		return sqlc.Wallet{}, err
	}

	wallet, err := s.repo.Create(
		ctx,
		organizationID,
		currency,
	)
	if err == nil {
		return wallet, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewInternal(
			"create billing wallet",
			err,
		)
	}

	wallet, err = s.repo.Get(
		ctx,
		organizationID,
		currency,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound(
			"organization or wallet not found",
		)
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal(
			"get billing wallet after create",
			err,
		)
	}

	return wallet, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	currency string,
) (sqlc.Wallet, error) {
	if organizationID == uuid.Nil {
		return sqlc.Wallet{}, apperror.NewBadRequest(
			"organization context required",
		)
	}

	currency, err := normalizeCurrency(currency)
	if err != nil {
		return sqlc.Wallet{}, err
	}

	wallet, err := s.repo.Get(
		ctx,
		organizationID,
		currency,
	)

	return wallet, walletReadError(err)
}

func (s *Service) GetByID(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Wallet, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return sqlc.Wallet{}, apperror.NewBadRequest(
			"organization and wallet are required",
		)
	}

	wallet, err := s.repo.GetByID(
		ctx,
		organizationID,
		id,
	)

	return wallet, walletReadError(err)
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Wallet, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest(
			"organization context required",
		)
	}

	wallets, err := s.repo.List(
		ctx,
		organizationID,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list billing wallets",
			err,
		)
	}

	return wallets, nil
}

func (s *Service) SetStatus(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	status string,
) (sqlc.Wallet, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return sqlc.Wallet{}, apperror.NewBadRequest(
			"organization and wallet are required",
		)
	}

	status, err := normalizeStatus(status)
	if err != nil {
		return sqlc.Wallet{}, err
	}

	wallet, err := s.repo.SetStatus(
		ctx,
		organizationID,
		id,
		status,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Wallet{}, apperror.NewNotFound(
			"wallet not found",
		)
	}
	if err != nil {
		return sqlc.Wallet{}, apperror.NewInternal(
			"update billing wallet status",
			err,
		)
	}

	return wallet, nil
}

func (s *Service) EventByOperationID(
	ctx context.Context,
	operationID uuid.UUID,
) (sqlc.WalletEvent, error) {
	if operationID == uuid.Nil {
		return sqlc.WalletEvent{}, apperror.NewBadRequest(
			"operation id is required",
		)
	}

	event, err := s.repo.EventByOperationID(
		ctx,
		operationID,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.WalletEvent{}, apperror.NewNotFound(
			"wallet event not found",
		)
	}
	if err != nil {
		return sqlc.WalletEvent{}, apperror.NewInternal(
			"get wallet event",
			err,
		)
	}

	return event, nil
}

func (s *Service) ListEvents(
	ctx context.Context,
	organizationID uuid.UUID,
	walletID uuid.UUID,
	limit int32,
) ([]sqlc.WalletEvent, error) {
	if organizationID == uuid.Nil || walletID == uuid.Nil {
		return nil, apperror.NewBadRequest(
			"organization and wallet are required",
		)
	}

	limit, err := normalizeEventLimit(limit)
	if err != nil {
		return nil, err
	}

	events, err := s.repo.ListEvents(
		ctx,
		organizationID,
		walletID,
		limit,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list wallet events",
			err,
		)
	}

	return events, nil
}

func (s *Service) ListChargeEvents(
	ctx context.Context,
	organizationID uuid.UUID,
	chargeID uuid.UUID,
) ([]sqlc.WalletEvent, error) {
	if organizationID == uuid.Nil || chargeID == uuid.Nil {
		return nil, apperror.NewBadRequest(
			"organization and charge are required",
		)
	}

	events, err := s.repo.ListChargeEvents(
		ctx,
		organizationID,
		chargeID,
	)
	if err != nil {
		return nil, apperror.NewInternal(
			"list charge wallet events",
			err,
		)
	}

	return events, nil
}

func walletReadError(err error) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(
			"wallet not found",
		)
	}
	if err != nil {
		return apperror.NewInternal(
			"get billing wallet",
			err,
		)
	}

	return nil
}

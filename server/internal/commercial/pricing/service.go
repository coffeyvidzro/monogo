package pricing

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
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRateRequest,
) (Rate, error) {
	if err := normalizeCreateRateRequest(&req); err != nil {
		return Rate{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Create(
		ctx,
		sqlc.CreateCarrierRateParams{
			OrganizationID:    req.OrganizationID,
			DestinationPrefix: req.DestinationPrefix,
			Direction:         req.Direction,
			Currency:          req.Currency,
			RateMicros:        req.RateMicros,
			EffectiveAt:       pgconv.TimeToTimestamptz(req.EffectiveAt),
			ExpiresAt:         pgconv.NullableTimestamptz(req.ExpiresAt),
		},
	)
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) && pgErr.Code == "23505" {
		return Rate{}, apperror.NewConflict("carrier rate conflict")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return Rate{}, apperror.NewNotFound("carrier rate not found")
	}
	if err != nil {
		return Rate{}, apperror.NewInternal("create carrier rate", err)
	}

	return rateFromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	id uuid.UUID,
) (Rate, error) {
	if id == uuid.Nil {
		return Rate{}, apperror.NewNotFound("carrier rate not found")
	}

	row, err := s.repo.Get(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rate{}, apperror.NewNotFound("carrier rate not found")
	}
	if err != nil {
		return Rate{}, apperror.NewInternal("get carrier rate", err)
	}

	return rateFromRow(row), nil
}

func (s *Service) Resolve(
	ctx context.Context,
	req ResolveRequest,
) (Rate, error) {
	if err := normalizeResolveRequest(&req); err != nil {
		return Rate{}, apperror.NewBadRequest(err.Error())
	}
	if req.ResolvedAt.IsZero() {
		req.ResolvedAt = s.now().UTC()
	}

	row, err := s.repo.Resolve(
		ctx,
		sqlc.ResolveCarrierRateParams{
			OrganizationID:    req.OrganizationID,
			Direction:         req.Direction,
			Currency:          req.Currency,
			DestinationDigits: req.DestinationDigits,
			ResolvedAt:        pgconv.TimeToTimestamptz(req.ResolvedAt),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Rate{}, apperror.NewNotFound("carrier rate not found")
	}
	if err != nil {
		return Rate{}, apperror.NewInternal("resolve carrier rate", err)
	}

	return rateFromRow(row), nil
}

func rateFromRow(row sqlc.CarrierRate) Rate {
	return Rate{
		ID:                row.ID,
		OrganizationID:    row.OrganizationID,
		DestinationPrefix: row.DestinationPrefix,
		Direction:         row.Direction,
		Currency:          row.Currency,
		RateMicros:        row.RateMicros,
		EffectiveAt:       pgconv.TimestamptzToTime(row.EffectiveAt),
		ExpiresAt:         pgconv.TimestamptzToTimePtr(row.ExpiresAt),
		CreatedAt:         pgconv.TimestamptzToTime(row.CreatedAt),
	}
}

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

func (s *Service) CreateVoiceRate(
	ctx context.Context,
	req CreateVoiceRateRequest,
) (VoiceRate, error) {
	if err := normalizeCreateVoiceRateRequest(&req); err != nil {
		return VoiceRate{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.CreateVoiceRate(
		ctx,
		sqlc.CreateVoiceRateParams{
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
		return VoiceRate{}, apperror.NewConflict("voice rate conflict")
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return VoiceRate{}, apperror.NewNotFound("voice rate not found")
	}
	if err != nil {
		return VoiceRate{}, apperror.NewInternal("create voice rate", err)
	}

	return voiceRateFromRow(row), nil
}

func (s *Service) GetVoiceRate(
	ctx context.Context,
	id uuid.UUID,
) (VoiceRate, error) {
	if id == uuid.Nil {
		return VoiceRate{}, apperror.NewNotFound("voice rate not found")
	}

	row, err := s.repo.GetVoiceRate(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return VoiceRate{}, apperror.NewNotFound("voice rate not found")
	}
	if err != nil {
		return VoiceRate{}, apperror.NewInternal("get voice rate", err)
	}

	return voiceRateFromRow(row), nil
}

func (s *Service) ResolveVoiceRate(
	ctx context.Context,
	req ResolveVoiceRateRequest,
) (VoiceRate, error) {
	if err := normalizeResolveVoiceRateRequest(&req); err != nil {
		return VoiceRate{}, apperror.NewBadRequest(err.Error())
	}
	if req.ResolvedAt.IsZero() {
		req.ResolvedAt = s.now().UTC()
	}

	row, err := s.repo.ResolveVoiceRate(
		ctx,
		sqlc.ResolveVoiceRateParams{
			OrganizationID:    req.OrganizationID,
			Direction:         req.Direction,
			Currency:          req.Currency,
			DestinationDigits: req.DestinationDigits,
			ResolvedAt:        pgconv.TimeToTimestamptz(req.ResolvedAt),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return VoiceRate{}, apperror.NewNotFound("voice rate not found")
	}
	if err != nil {
		return VoiceRate{}, apperror.NewInternal("resolve voice rate", err)
	}

	return voiceRateFromRow(row), nil
}

func (s *Service) ResolveProductRate(
	ctx context.Context,
	req ResolveProductRateRequest,
) (ProductRate, error) {
	if err := normalizeResolveProductRateRequest(&req); err != nil {
		return ProductRate{}, apperror.NewBadRequest(err.Error())
	}
	if req.ResolvedAt.IsZero() {
		req.ResolvedAt = s.now().UTC()
	}

	row, err := s.repo.ResolveProductRate(
		ctx,
		sqlc.ResolveProductRateParams{
			OrganizationID: req.OrganizationID,
			Product:        req.Product,
			Selector:       req.Selector,
			Currency:       req.Currency,
			ResolvedAt:     pgconv.TimeToTimestamptz(req.ResolvedAt),
		},
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProductRate{}, apperror.NewNotFound("product rate not found")
	}
	if err != nil {
		return ProductRate{}, apperror.NewInternal("resolve product rate", err)
	}

	return productRateFromRow(row), nil
}

func voiceRateFromRow(row sqlc.VoiceRate) VoiceRate {
	return VoiceRate{
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

func productRateFromRow(row sqlc.ProductRate) ProductRate {
	return ProductRate{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Product:        row.Product,
		Selector:       row.Selector,
		Currency:       row.Currency,
		RateMicros:     row.RateMicros,
		EffectiveAt:    pgconv.TimestamptzToTime(row.EffectiveAt),
		ExpiresAt:      pgconv.TimestamptzToTimePtr(row.ExpiresAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
	}
}

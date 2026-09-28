package plans

import (
	"context"
	"errors"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) Create(
	ctx context.Context,
	req CreateRequest,
) (Plan, error) {
	if err := normalizeCreateRequest(&req); err != nil {
		return Plan{}, apperror.NewBadRequest(err.Error())
	}

	row, err := s.repo.Create(
		ctx,
		sqlc.CreateSubscriptionPlanParams{
			Code:         req.Code,
			Name:         req.Name,
			Currency:     req.Currency,
			AmountMicros: req.AmountMicros,
		},
	)
	if isUniqueViolation(err) {
		return Plan{}, apperror.NewConflict("plan conflict")
	}
	if err != nil {
		return Plan{}, apperror.NewInternal("create plan", err)
	}

	return fromRow(row), nil
}

func (s *Service) Get(
	ctx context.Context,
	id uuid.UUID,
) (Plan, error) {
	if id == uuid.Nil {
		return Plan{}, apperror.NewNotFound("plan not found")
	}

	row, err := s.repo.GetByID(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, apperror.NewNotFound("plan not found")
	}
	if err != nil {
		return Plan{}, apperror.NewInternal("get plan", err)
	}

	return fromRow(row), nil
}

func (s *Service) GetByCode(
	ctx context.Context,
	code string,
) (Plan, error) {
	code = normalizeCode(code)
	if !codePattern.MatchString(code) {
		return Plan{}, apperror.NewNotFound("plan not found")
	}

	row, err := s.repo.GetByCode(
		ctx,
		code,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, apperror.NewNotFound("plan not found")
	}
	if err != nil {
		return Plan{}, apperror.NewInternal("get plan by code", err)
	}

	return fromRow(row), nil
}

func (s *Service) List(
	ctx context.Context,
) ([]Plan, error) {
	rows, err := s.repo.List(ctx)
	if err != nil {
		return nil, apperror.NewInternal("list plans", err)
	}

	result := make([]Plan, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			fromRow(row),
		)
	}

	return result, nil
}

func (s *Service) Archive(
	ctx context.Context,
	id uuid.UUID,
) (Plan, error) {
	if id == uuid.Nil {
		return Plan{}, apperror.NewNotFound("plan not found")
	}

	row, err := s.repo.Archive(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Plan{}, apperror.NewNotFound("plan not found")
	}
	if err != nil {
		return Plan{}, apperror.NewInternal("archive plan", err)
	}

	return fromRow(row), nil
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

func fromRow(row sqlc.SubscriptionPlan) Plan {
	return Plan{
		ID:           row.ID,
		Code:         row.Code,
		Name:         row.Name,
		Currency:     row.Currency,
		Interval:     row.Interval,
		AmountMicros: row.AmountMicros,
		Status:       row.Status,
		CreatedAt:    pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:    pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

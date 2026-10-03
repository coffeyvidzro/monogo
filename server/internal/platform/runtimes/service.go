package runtimes

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (Runtime, error) {
	if organizationID == uuid.Nil {
		return Runtime{}, apperror.NewBadRequest("organization_id is required")
	}
	if err := normalizeCreate(&req); err != nil {
		return Runtime{}, err
	}
	value, err := s.repo.Create(ctx, organizationID, req)
	if runtimeConflict(err) {
		return Runtime{}, apperror.NewConflict("runtime name already exists")
	}
	if err != nil {
		return Runtime{}, apperror.NewInternal("create runtime", err)
	}
	return value, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (Runtime, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Runtime{}, apperror.NewBadRequest("organization and runtime ids are required")
	}
	value, err := s.repo.Get(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Runtime{}, apperror.NewNotFound("runtime not found")
	}
	if err != nil {
		return Runtime{}, apperror.NewInternal("get runtime", err)
	}
	return value, nil
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Runtime, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list runtimes", err)
	}
	return values, nil
}

func (s *Service) Heartbeat(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	req HeartbeatRequest,
) (Runtime, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Runtime{}, apperror.NewBadRequest("organization and runtime ids are required")
	}
	if err := normalizeHeartbeat(&req); err != nil {
		return Runtime{}, err
	}
	value, err := s.repo.Heartbeat(ctx, organizationID, id, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return Runtime{}, apperror.NewNotFound("active runtime not found")
	}
	if err != nil {
		return Runtime{}, apperror.NewInternal("heartbeat runtime", err)
	}
	return value, nil
}

func runtimeConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

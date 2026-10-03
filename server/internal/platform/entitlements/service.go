package entitlements

import (
	"context"
	"errors"
	"fmt"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Entitlement, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list organization entitlements", err)
	}
	return values, nil
}

func (s *Service) Enabled(
	ctx context.Context,
	organizationID uuid.UUID,
	capability Capability,
) (bool, error) {
	if organizationID == uuid.Nil {
		return false, apperror.NewBadRequest("organization_id is required")
	}
	capability, err := normalizeCapability(capability)
	if err != nil {
		return false, err
	}

	value, err := s.repo.Get(ctx, organizationID, capability)
	if errors.Is(err, pgx.ErrNoRows) {
		return false, nil
	}
	if err != nil {
		return false, apperror.NewInternal("get organization entitlement", err)
	}
	return value.Enabled, nil
}

func (s *Service) Require(
	ctx context.Context,
	organizationID uuid.UUID,
	capability Capability,
) error {
	enabled, err := s.Enabled(ctx, organizationID, capability)
	if err != nil {
		return err
	}
	if !enabled {
		return apperror.NewForbidden(
			fmt.Sprintf("organization capability %q is not enabled", capability),
		)
	}
	return nil
}

func (s *Service) Set(
	ctx context.Context,
	organizationID uuid.UUID,
	capability Capability,
	enabled bool,
) (Entitlement, error) {
	if organizationID == uuid.Nil {
		return Entitlement{}, apperror.NewBadRequest("organization_id is required")
	}
	capability, err := normalizeCapability(capability)
	if err != nil {
		return Entitlement{}, err
	}

	value, err := s.repo.Set(ctx, organizationID, capability, enabled)
	if err != nil {
		return Entitlement{}, apperror.NewInternal("set organization entitlement", err)
	}
	return value, nil
}

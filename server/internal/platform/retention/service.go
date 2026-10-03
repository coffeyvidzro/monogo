package retention

import (
	"context"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]Policy, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list retention policies", err)
	}
	return values, nil
}
func (s *Service) Upsert(ctx context.Context, organizationID uuid.UUID, resource string, req UpsertRequest) (Policy, error) {
	if organizationID == uuid.Nil {
		return Policy{}, apperror.NewBadRequest("organization_id is required")
	}
	if err := validateResource(resource); err != nil {
		return Policy{}, err
	}
	if err := validateUpsert(req); err != nil {
		return Policy{}, err
	}
	value, err := s.repo.Upsert(ctx, organizationID, resource, req)
	if err != nil {
		return Policy{}, apperror.NewInternal("upsert retention policy", err)
	}
	return value, nil
}
func (s *Service) Delete(ctx context.Context, organizationID uuid.UUID, resource string) error {
	if organizationID == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	if err := validateResource(resource); err != nil {
		return err
	}
	if err := s.repo.Delete(ctx, organizationID, resource); err != nil {
		return apperror.NewInternal("delete retention policy", err)
	}
	return nil
}

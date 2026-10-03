package networking

import (
	"context"
	"net/netip"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}
func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (Policy, error) {
	if organizationID == uuid.Nil {
		return Policy{}, apperror.NewBadRequest("organization_id is required")
	}
	req, prefix, err := validateCreate(req)
	if err != nil {
		return Policy{}, err
	}
	value, err := s.repo.Create(ctx, organizationID, req, prefix)
	if err != nil {
		return Policy{}, apperror.NewInternal("create network policy", err)
	}
	return value, nil
}
func (s *Service) List(ctx context.Context, organizationID uuid.UUID) ([]Policy, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list network policies", err)
	}
	return values, nil
}
func (s *Service) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest) (Policy, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Policy{}, apperror.NewBadRequest("organization and policy ids are required")
	}
	var prefix *netip.Prefix
	if req.SourceCIDR != nil {
		value, err := netip.ParsePrefix(strings.TrimSpace(*req.SourceCIDR))
		if err != nil {
			return Policy{}, apperror.NewBadRequest("source_cidr must be a valid CIDR")
		}
		value = value.Masked()
		prefix = &value
	}
	value, err := s.repo.Update(ctx, organizationID, id, req, prefix)
	if err != nil {
		return Policy{}, apperror.NewInternal("update network policy", err)
	}
	return value, nil
}
func (s *Service) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return apperror.NewBadRequest("organization and policy ids are required")
	}
	if err := s.repo.Delete(ctx, organizationID, id); err != nil {
		return apperror.NewInternal("delete network policy", err)
	}
	return nil
}

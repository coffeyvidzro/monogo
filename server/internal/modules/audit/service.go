package audit

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service { return &Service{repo: repo} }

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, limit, offset int32) ([]Event, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization id is required")
	}
	if limit < 1 || limit > 100 {
		return nil, apperror.NewBadRequest("limit must be between 1 and 100")
	}
	if offset < 0 {
		return nil, apperror.NewBadRequest("offset cannot be negative")
	}
	items, err := s.repo.List(ctx, organizationID, limit, offset)
	if err != nil {
		return nil, apperror.NewInternal("list audit events", err)
	}
	return items, nil
}

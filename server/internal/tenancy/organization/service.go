package organization

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

const (
	ownerRole = "owner"
	adminRole = "admin"
)

type Service struct {
	repo serviceRepository
}

type serviceRepository interface {
	CreateWithOwner(context.Context, string, uuid.UUID) (sqlc.Organization, error)
	GetByID(context.Context, uuid.UUID) (sqlc.Organization, error)
	GetMember(context.Context, sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error)
	Update(context.Context, sqlc.UpdateOrganizationParams) (sqlc.Organization, error)
	Delete(context.Context, uuid.UUID) error
	ListByUserID(context.Context, uuid.UUID) ([]sqlc.ListOrganizationsByUserIDRow, error)
}

func NewService(repo serviceRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Create(ctx context.Context, userID uuid.UUID, req CreateRequest) (Response, error) {
	if userID == uuid.Nil {
		return Response{}, apperror.NewUnauthorized("authentication required")
	}

	name, err := normalizeRequiredString(req.Name, "name")
	if err != nil {
		return Response{}, err
	}

	org, err := s.repo.CreateWithOwner(ctx, name, userID)
	if err != nil {
		return Response{}, apperror.NewInternal("create organization", err)
	}

	return toResponse(org, ownerRole), nil
}

func (s *Service) Get(ctx context.Context, userID, orgID uuid.UUID) (Response, error) {
	if err := s.requireMember(ctx, userID, orgID); err != nil {
		return Response{}, err
	}

	org, err := s.repo.GetByID(ctx, orgID)
	if err != nil {
		return Response{}, apperror.NewNotFound("organization not found")
	}

	member, err := s.repo.GetMember(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: orgID, UserID: userID})
	if err != nil {
		return Response{}, apperror.NewNotFound("organization not found")
	}

	return toResponse(org, member.Role), nil
}

func (s *Service) Update(ctx context.Context, userID, orgID uuid.UUID, req UpdateRequest) (Response, error) {
	member, err := s.requireRole(ctx, userID, orgID, ownerRole, adminRole)
	if err != nil {
		return Response{}, err
	}

	if req.Name != nil {
		name, err := normalizeRequiredString(*req.Name, "name")
		if err != nil {
			return Response{}, err
		}
		req.Name = &name
	}

	org, err := s.repo.Update(ctx, sqlc.UpdateOrganizationParams{ID: orgID, Name: req.Name})
	if err != nil {
		return Response{}, apperror.NewNotFound("organization not found")
	}

	return toResponse(org, member.Role), nil
}

func (s *Service) Delete(ctx context.Context, userID, orgID uuid.UUID) error {
	if _, err := s.requireRole(ctx, userID, orgID, ownerRole); err != nil {
		return err
	}

	return s.repo.Delete(ctx, orgID)
}

func (s *Service) requireRole(ctx context.Context, userID, orgID uuid.UUID, roles ...string) (sqlc.OrganizationMember, error) {
	if userID == uuid.Nil {
		return sqlc.OrganizationMember{}, apperror.NewUnauthorized("authentication required")
	}
	if err := validateOrganizationID(orgID); err != nil {
		return sqlc.OrganizationMember{}, err
	}

	member, err := s.repo.GetMember(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: orgID, UserID: userID})
	if err != nil {
		return sqlc.OrganizationMember{}, apperror.NewNotFound("organization not found")
	}
	for _, role := range roles {
		if member.Role == role {
			return member, nil
		}
	}
	return sqlc.OrganizationMember{}, apperror.NewForbidden("organization role does not permit this action")
}

func (s *Service) List(ctx context.Context, userID uuid.UUID) ([]Response, error) {
	if userID == uuid.Nil {
		return nil, apperror.NewUnauthorized("authentication required")
	}

	orgs, err := s.repo.ListByUserID(ctx, userID)
	if err != nil {
		return nil, err
	}

	responses := make([]Response, 0, len(orgs))
	for _, org := range orgs {
		responses = append(responses, Response{
			ID:        org.ID,
			Name:      org.Name,
			Status:    org.Status,
			Role:      org.MemberRole,
			CreatedAt: pgconv.TimestamptzToTime(org.CreatedAt),
			UpdatedAt: pgconv.TimestamptzToTime(org.UpdatedAt),
		})
	}

	return responses, nil
}

func (s *Service) requireMember(ctx context.Context, userID, orgID uuid.UUID) error {
	if userID == uuid.Nil {
		return apperror.NewUnauthorized("authentication required")
	}
	if err := validateOrganizationID(orgID); err != nil {
		return err
	}

	if _, err := s.repo.GetMember(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: orgID, UserID: userID}); err != nil {
		return apperror.NewNotFound("organization not found")
	}

	return nil
}

func toResponse(org sqlc.Organization, role string) Response {
	return Response{
		ID:        org.ID,
		Name:      org.Name,
		Status:    org.Status,
		Role:      role,
		CreatedAt: pgconv.TimestamptzToTime(org.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(org.UpdatedAt),
	}
}

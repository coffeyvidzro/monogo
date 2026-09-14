package members

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgconn"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/apperror"
)

type Service struct {
	repo serviceRepository
}

type serviceRepository interface {
	Add(context.Context, sqlc.AddOrganizationMemberParams) (sqlc.OrganizationMember, error)
	Get(context.Context, sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error)
	ListByOrganizationID(context.Context, uuid.UUID) ([]sqlc.OrganizationMember, error)
	UpdateRole(context.Context, sqlc.UpdateMemberRoleParams) (sqlc.OrganizationMember, error)
	Disable(context.Context, sqlc.DisableOrganizationMemberParams) error
}

func NewService(repo serviceRepository) *Service {
	return &Service{repo: repo}
}

func (s *Service) Add(ctx context.Context, requesterID, organizationID uuid.UUID, req CreateRequest) (sqlc.OrganizationMember, error) {
	if err := s.requireAdmin(ctx, requesterID, organizationID); err != nil {
		return sqlc.OrganizationMember{}, err
	}
	if err := validateUserID(req.UserID, "user_id"); err != nil {
		return sqlc.OrganizationMember{}, err
	}

	role, err := normalizeRole(req.Role)
	if err != nil {
		return sqlc.OrganizationMember{}, err
	}
	if role == roleOwner {
		return sqlc.OrganizationMember{}, apperror.NewConflict("organization owner must be transferred explicitly")
	}

	member, err := s.repo.Add(ctx, sqlc.AddOrganizationMemberParams{
		OrganizationID: organizationID,
		UserID:         req.UserID,
		Role:           role,
		ActorUserID:    requesterID,
	})
	if err != nil {
		if isUniqueViolation(err) {
			return sqlc.OrganizationMember{}, apperror.NewConflict("organization member already exists")
		}
		return sqlc.OrganizationMember{}, apperror.NewNotFound("user not found")
	}

	return member, nil
}

func (s *Service) Get(ctx context.Context, requesterID, organizationID, memberID uuid.UUID) (sqlc.OrganizationMember, error) {
	if err := s.requireMember(ctx, requesterID, organizationID); err != nil {
		return sqlc.OrganizationMember{}, err
	}
	if err := validateUserID(memberID, "user_id"); err != nil {
		return sqlc.OrganizationMember{}, err
	}

	member, err := s.repo.Get(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: organizationID, UserID: memberID})
	if err != nil {
		return sqlc.OrganizationMember{}, apperror.NewNotFound("organization member not found")
	}

	return member, nil
}

func (s *Service) List(ctx context.Context, requesterID, organizationID uuid.UUID) ([]sqlc.OrganizationMember, error) {
	if err := s.requireMember(ctx, requesterID, organizationID); err != nil {
		return nil, err
	}

	return s.repo.ListByOrganizationID(ctx, organizationID)
}

func (s *Service) Update(ctx context.Context, requesterID, organizationID, memberID uuid.UUID, req UpdateRequest) (sqlc.OrganizationMember, error) {
	if err := s.requireAdmin(ctx, requesterID, organizationID); err != nil {
		return sqlc.OrganizationMember{}, err
	}
	if err := validateUserID(memberID, "user_id"); err != nil {
		return sqlc.OrganizationMember{}, err
	}

	role, err := normalizeRole(req.Role)
	if err != nil {
		return sqlc.OrganizationMember{}, err
	}
	if role == roleOwner {
		return sqlc.OrganizationMember{}, apperror.NewConflict("organization owner must be transferred explicitly")
	}

	current, err := s.repo.Get(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: organizationID, UserID: memberID})
	if err != nil {
		return sqlc.OrganizationMember{}, apperror.NewNotFound("organization member not found")
	}
	if current.Role == roleOwner {
		return sqlc.OrganizationMember{}, apperror.NewConflict("organization owner cannot be demoted")
	}

	member, err := s.repo.UpdateRole(ctx, sqlc.UpdateMemberRoleParams{
		OrganizationID: organizationID,
		UserID:         memberID,
		Role:           role,
		ActorUserID:    requesterID,
	})
	if err != nil {
		return sqlc.OrganizationMember{}, apperror.NewNotFound("organization member not found")
	}

	return member, nil
}

func (s *Service) Delete(ctx context.Context, requesterID, organizationID, memberID uuid.UUID) error {
	if err := s.requireAdmin(ctx, requesterID, organizationID); err != nil {
		return err
	}
	if err := validateUserID(memberID, "user_id"); err != nil {
		return err
	}

	member, err := s.repo.Get(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: organizationID, UserID: memberID})
	if err != nil {
		return apperror.NewNotFound("organization member not found")
	}
	if member.Role == roleOwner {
		return apperror.NewConflict("organization owner cannot be removed")
	}

	return s.repo.Disable(ctx, sqlc.DisableOrganizationMemberParams{
		OrganizationID: organizationID,
		UserID:         memberID,
		ActorUserID:    requesterID,
	})
}

func (s *Service) requireMember(ctx context.Context, requesterID, organizationID uuid.UUID) error {
	if err := validateUserID(requesterID, "requester_id"); err != nil {
		return apperror.NewUnauthorized("authentication required")
	}
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}

	if _, err := s.repo.Get(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: organizationID, UserID: requesterID}); err != nil {
		return apperror.NewNotFound("organization not found")
	}

	return nil
}

func (s *Service) requireAdmin(ctx context.Context, requesterID, organizationID uuid.UUID) error {
	if err := validateUserID(requesterID, "requester_id"); err != nil {
		return apperror.NewUnauthorized("authentication required")
	}
	if err := validateOrganizationID(organizationID); err != nil {
		return err
	}

	member, err := s.repo.Get(ctx, sqlc.GetOrganizationMemberParams{OrganizationID: organizationID, UserID: requesterID})
	if err != nil {
		return apperror.NewNotFound("organization not found")
	}
	if member.Role != roleOwner && member.Role != roleAdmin {
		return apperror.NewForbidden("organization owner or admin role required")
	}

	return nil
}

func toResponse(member sqlc.OrganizationMember) Response {
	return Response{
		UserID:    member.UserID,
		Role:      member.Role,
		Status:    member.Status,
		CreatedAt: pgconv.TimestamptzToTime(member.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(member.UpdatedAt),
	}
}

func toResponses(members []sqlc.OrganizationMember) []Response {
	responses := make([]Response, 0, len(members))
	for _, member := range members {
		responses = append(responses, toResponse(member))
	}
	return responses
}

func isUniqueViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

package scim

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type Principal struct {
	TokenID        uuid.UUID
	OrganizationID uuid.UUID
}

type Service struct{ repo *Repository }

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
	}
}

func (s *Service) CreateToken(ctx context.Context, organizationID uuid.UUID, req CreateTokenRequest) (CreatedToken, error) {
	if organizationID == uuid.Nil {
		return CreatedToken{}, apperror.NewBadRequest("organization_id is required")
	}
	req, err := normalizeCreateToken(req, time.Now().UTC())
	if err != nil {
		return CreatedToken{}, err
	}
	secret, prefix, hash, err := generateToken()
	if err != nil {
		return CreatedToken{}, apperror.NewInternal("generate SCIM token", err)
	}
	value, err := s.repo.Create(ctx, organizationID, req, prefix, hash)
	if err != nil {
		return CreatedToken{}, apperror.NewInternal("create SCIM token", err)
	}
	return CreatedToken{
		Token:  value,
		Secret: secret,
	}, nil
}
func (s *Service) ListTokens(ctx context.Context, organizationID uuid.UUID) ([]Token, error) {
	if organizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	values, err := s.repo.List(ctx, organizationID)
	if err != nil {
		return nil, apperror.NewInternal("list SCIM tokens", err)
	}
	return values, nil
}
func (s *Service) RevokeToken(ctx context.Context, organizationID, id uuid.UUID) (Token, error) {
	if organizationID == uuid.Nil || id == uuid.Nil {
		return Token{}, apperror.NewBadRequest("organization and token ids are required")
	}
	value, err := s.repo.Revoke(ctx, organizationID, id)
	if errors.Is(err, pgx.ErrNoRows) {
		return Token{}, apperror.NewNotFound("SCIM token not found")
	}
	if err != nil {
		return Token{}, apperror.NewInternal("revoke SCIM token", err)
	}
	return value, nil
}
func (s *Service) Authenticate(ctx context.Context, secret string) (Principal, error) {
	if len(secret) <= len(tokenPrefix) || secret[:len(tokenPrefix)] != tokenPrefix {
		return Principal{}, apperror.NewUnauthorized("invalid SCIM token")
	}
	row, err := s.repo.Authenticate(ctx, hashToken(secret))
	if err != nil {
		return Principal{}, apperror.NewUnauthorized("invalid SCIM token")
	}
	if err := s.repo.Touch(ctx, row.OrganizationID, row.ID); err != nil {
		return Principal{}, apperror.NewInternal("update SCIM token usage", err)
	}
	return Principal{
		TokenID:        row.ID,
		OrganizationID: row.OrganizationID,
	}, nil
}

func (s *Service) CreateUser(ctx context.Context, organizationID uuid.UUID, input UserInput) (User, error) {
	if organizationID == uuid.Nil {
		return User{}, apperror.NewBadRequest("organization_id is required")
	}
	input, active, err := normalizeUserInput(input)
	if err != nil {
		return User{}, err
	}
	id := uuid.New()
	externalID := input.ExternalID
	if externalID == "" {
		externalID = id.String()
	}
	record, err := s.repo.CreateUser(ctx, organizationID, id, input, externalID, active)
	if scimConflict(err) {
		return User{}, apperror.NewConflict("SCIM user already exists")
	}
	if err != nil {
		return User{}, apperror.NewInternal("create SCIM user", err)
	}
	return userResource(record), nil
}

func (s *Service) GetUser(ctx context.Context, organizationID, id uuid.UUID) (User, error) {
	record, err := s.repo.GetUser(ctx, organizationID, id)
	if err != nil {
		return User{}, scimDatabaseError(err, "SCIM user not found")
	}
	return userResource(record), nil
}

func (s *Service) ListUsers(ctx context.Context, organizationID uuid.UUID, startIndex, count int, filter string) (ListResponse, error) {
	startIndex, limit, offset := page(startIndex, count)
	if userName, ok, err := equalityFilter(filter, "userName"); err != nil {
		return ListResponse{}, err
	} else if ok {
		record, err := s.repo.GetUserByUserName(ctx, organizationID, strings.ToLower(userName))
		if errors.Is(err, pgx.ErrNoRows) {
			return listResponse(startIndex, []User{}, 0), nil
		}
		if err != nil {
			return ListResponse{}, apperror.NewInternal("list SCIM users", err)
		}
		return listResponse(startIndex, []User{userResource(record)}, 1), nil
	}
	records, total, err := s.repo.ListUsers(ctx, organizationID, limit, offset)
	if err != nil {
		return ListResponse{}, apperror.NewInternal("list SCIM users", err)
	}
	resources := make([]User, 0, len(records))
	for _, record := range records {
		resources = append(resources, userResource(record))
	}
	return listResponse(startIndex, resources, total), nil
}

func (s *Service) ReplaceUser(ctx context.Context, organizationID, id uuid.UUID, input UserInput) (User, error) {
	input, active, err := normalizeUserInput(input)
	if err != nil {
		return User{}, err
	}
	record, err := s.repo.ReplaceUser(ctx, organizationID, id, input, active)
	if scimConflict(err) {
		return User{}, apperror.NewConflict("SCIM userName is already in use")
	}
	if err != nil {
		return User{}, scimDatabaseError(err, "SCIM user not found")
	}
	return userResource(record), nil
}

func (s *Service) DeleteUser(ctx context.Context, organizationID, id uuid.UUID) error {
	if err := s.repo.DeleteUser(ctx, organizationID, id); err != nil {
		return scimDatabaseError(err, "SCIM user not found or cannot be deprovisioned")
	}
	return nil
}

func (s *Service) CreateGroup(ctx context.Context, organizationID uuid.UUID, input GroupInput) (Group, error) {
	input, memberIDs, err := normalizeGroupInput(input)
	if err != nil {
		return Group{}, err
	}
	if err := s.validateMembers(ctx, organizationID, memberIDs); err != nil {
		return Group{}, err
	}
	record, err := s.repo.CreateGroup(ctx, organizationID, uuid.New(), input)
	if scimConflict(err) {
		return Group{}, apperror.NewConflict("SCIM group already exists")
	}
	if err != nil {
		return Group{}, apperror.NewInternal("create SCIM group", err)
	}
	if _, err := s.repo.ReplaceGroupMembers(ctx, organizationID, record.ID, memberIDs); err != nil {
		return Group{}, apperror.NewInternal("set SCIM group members", err)
	}
	return s.groupResource(ctx, record)
}

func (s *Service) GetGroup(ctx context.Context, organizationID, id uuid.UUID) (Group, error) {
	record, err := s.repo.GetGroup(ctx, organizationID, id)
	if err != nil {
		return Group{}, scimDatabaseError(err, "SCIM group not found")
	}
	return s.groupResource(ctx, record)
}

func (s *Service) ListGroups(ctx context.Context, organizationID uuid.UUID, startIndex, count int, filter string) (ListResponse, error) {
	startIndex, limit, offset := page(startIndex, count)
	if displayName, ok, err := equalityFilter(filter, "displayName"); err != nil {
		return ListResponse{}, err
	} else if ok {
		record, err := s.repo.GetGroupByDisplayName(ctx, organizationID, displayName)
		if errors.Is(err, pgx.ErrNoRows) {
			return listResponse(startIndex, []Group{}, 0), nil
		}
		if err != nil {
			return ListResponse{}, apperror.NewInternal("list SCIM groups", err)
		}
		resource, err := s.groupResource(ctx, record)
		if err != nil {
			return ListResponse{}, err
		}
		return listResponse(startIndex, []Group{resource}, 1), nil
	}
	records, total, err := s.repo.ListGroups(ctx, organizationID, limit, offset)
	if err != nil {
		return ListResponse{}, apperror.NewInternal("list SCIM groups", err)
	}
	resources := make([]Group, 0, len(records))
	for _, record := range records {
		resource, err := s.groupResource(ctx, record)
		if err != nil {
			return ListResponse{}, err
		}
		resources = append(resources, resource)
	}
	return listResponse(startIndex, resources, total), nil
}

func (s *Service) ReplaceGroup(ctx context.Context, organizationID, id uuid.UUID, input GroupInput) (Group, error) {
	input, memberIDs, err := normalizeGroupInput(input)
	if err != nil {
		return Group{}, err
	}
	if err := s.validateMembers(ctx, organizationID, memberIDs); err != nil {
		return Group{}, err
	}
	record, err := s.repo.ReplaceGroup(ctx, organizationID, id, input)
	if scimConflict(err) {
		return Group{}, apperror.NewConflict("SCIM group name or externalId is already in use")
	}
	if err != nil {
		return Group{}, scimDatabaseError(err, "SCIM group not found")
	}
	matched, err := s.repo.ReplaceGroupMembers(ctx, organizationID, id, memberIDs)
	if err != nil {
		return Group{}, apperror.NewInternal("replace SCIM group members", err)
	}
	if len(matched) != len(memberIDs) {
		return Group{}, apperror.NewBadRequest("one or more SCIM group members do not belong to this organization")
	}
	return s.groupResource(ctx, record)
}

func (s *Service) DeleteGroup(ctx context.Context, organizationID, id uuid.UUID) error {
	if _, err := s.repo.GetGroup(ctx, organizationID, id); err != nil {
		return scimDatabaseError(err, "SCIM group not found")
	}
	if err := s.repo.DeleteGroup(ctx, organizationID, id); err != nil {
		return apperror.NewInternal("delete SCIM group", err)
	}
	return nil
}

func (s *Service) validateMembers(ctx context.Context, organizationID uuid.UUID, memberIDs []uuid.UUID) error {
	for _, id := range memberIDs {
		if _, err := s.repo.GetUser(ctx, organizationID, id); err != nil {
			if errors.Is(err, pgx.ErrNoRows) {
				return apperror.NewBadRequest(fmt.Sprintf("SCIM group member %s does not belong to this organization", id))
			}
			return apperror.NewInternal("validate SCIM group member", err)
		}
	}
	return nil
}

func (s *Service) groupResource(ctx context.Context, record GroupRecord) (Group, error) {
	members, err := s.repo.ListGroupMembers(ctx, record.OrganizationID, record.ID)
	if err != nil {
		return Group{}, apperror.NewInternal("list SCIM group members", err)
	}
	externalID := ""
	if record.ExternalID != nil {
		externalID = *record.ExternalID
	}
	return Group{
		Schemas: []string{groupSchema}, ID: record.ID.String(), ExternalID: externalID,
		DisplayName: record.DisplayName, Members: members,
		Meta: Meta{ResourceType: "Group", Created: record.CreatedAt, LastModified: record.UpdatedAt},
	}, nil
}

func userResource(record UserRecord) User {
	displayName := ""
	if record.DisplayName != nil {
		displayName = *record.DisplayName
	}
	return User{
		Schemas: []string{userSchema}, ID: record.ID.String(), ExternalID: record.ExternalID,
		UserName: record.UserName, DisplayName: displayName, Active: record.Active,
		Meta: Meta{ResourceType: "User", Created: record.CreatedAt, LastModified: record.UpdatedAt},
	}
}

func listResponse[T any](startIndex int, resources []T, total int64) ListResponse {
	return ListResponse{
		Schemas: []string{listSchema}, TotalResults: total, StartIndex: startIndex,
		ItemsPerPage: len(resources), Resources: resources,
	}
}

func equalityFilter(filter, attribute string) (string, bool, error) {
	filter = strings.TrimSpace(filter)
	if filter == "" {
		return "", false, nil
	}
	parts := strings.SplitN(filter, " eq ", 2)
	if len(parts) != 2 || !strings.EqualFold(strings.TrimSpace(parts[0]), attribute) {
		return "", false, apperror.NewBadRequest("unsupported SCIM filter")
	}
	value := strings.TrimSpace(parts[1])
	if len(value) < 2 || value[0] != '"' || value[len(value)-1] != '"' {
		return "", false, apperror.NewBadRequest("SCIM filter value must be quoted")
	}
	return value[1 : len(value)-1], true, nil
}

func scimDatabaseError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound(message)
	}
	return apperror.NewInternal(message, err)
}

func scimConflict(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23505"
}

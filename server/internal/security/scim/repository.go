package scim

import (
	"context"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) Create(ctx context.Context, organizationID uuid.UUID, req CreateTokenRequest, prefix, hash string) (Token, error) {
	row, err := r.queries.CreateSCIMToken(ctx, sqlc.CreateSCIMTokenParams{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           req.Name,
		TokenPrefix:    prefix,
		TokenHash:      hash,
		ExpiresAt:      timestamp(req.ExpiresAt),
	})
	if err != nil {
		return Token{}, err
	}
	return tokenFromCreate(row), nil
}
func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]Token, error) {
	rows, err := r.queries.ListSCIMTokens(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]Token, 0, len(rows))
	for _, row := range rows {
		out = append(out, tokenFromList(row))
	}
	return out, nil
}
func (r *Repository) Authenticate(ctx context.Context, hash string) (sqlc.ScimToken, error) {
	return r.queries.GetActiveSCIMTokenByHash(ctx, hash)
}
func (r *Repository) Touch(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.TouchSCIMToken(ctx, sqlc.TouchSCIMTokenParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}
func (r *Repository) Revoke(ctx context.Context, organizationID, id uuid.UUID) (Token, error) {
	row, err := r.queries.RevokeSCIMToken(ctx, sqlc.RevokeSCIMTokenParams{
		ID:             id,
		OrganizationID: organizationID,
	})
	if err != nil {
		return Token{}, err
	}
	return tokenFromRevoke(row), nil
}

func (r *Repository) CreateUser(ctx context.Context, organizationID uuid.UUID, id uuid.UUID, input UserInput, externalID string, active bool) (UserRecord, error) {
	row, err := r.queries.CreateSCIMUser(ctx, sqlc.CreateSCIMUserParams{
		UserName:       input.UserName,
		DisplayName:    nullableString(input.DisplayName),
		OrganizationID: organizationID,
		Active:         active,
		ID:             id,
		ExternalID:     externalID,
	})
	if err != nil {
		return UserRecord{}, err
	}
	return userRecord(row.ID, row.OrganizationID, row.UserID, row.ExternalID, row.UserName, row.DisplayName, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetUser(ctx context.Context, organizationID, id uuid.UUID) (UserRecord, error) {
	row, err := r.queries.GetSCIMUser(ctx, sqlc.GetSCIMUserParams{
		OrganizationID: organizationID,
		ID:             id,
	})
	if err != nil {
		return UserRecord{}, err
	}
	return userRecord(row.ID, row.OrganizationID, row.UserID, row.ExternalID, row.UserName, row.DisplayName, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) GetUserByUserName(ctx context.Context, organizationID uuid.UUID, userName string) (UserRecord, error) {
	row, err := r.queries.GetSCIMUserByUserName(ctx, sqlc.GetSCIMUserByUserNameParams{
		OrganizationID: organizationID,
		UserName:       userName,
	})
	if err != nil {
		return UserRecord{}, err
	}
	return userRecord(row.ID, row.OrganizationID, row.UserID, row.ExternalID, row.UserName, row.DisplayName, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) ListUsers(ctx context.Context, organizationID uuid.UUID, limit, offset int32) ([]UserRecord, int64, error) {
	rows, err := r.queries.ListSCIMUsers(ctx, sqlc.ListSCIMUsersParams{
		OrganizationID: organizationID,
		LimitCount:     limit,
		OffsetCount:    offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountSCIMUsers(ctx, organizationID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]UserRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, userRecord(row.ID, row.OrganizationID, row.UserID, row.ExternalID, row.UserName, row.DisplayName, row.Active, row.CreatedAt, row.UpdatedAt))
	}
	return out, total, nil
}

func (r *Repository) ReplaceUser(ctx context.Context, organizationID, id uuid.UUID, input UserInput, active bool) (UserRecord, error) {
	row, err := r.queries.ReplaceSCIMUser(ctx, sqlc.ReplaceSCIMUserParams{
		OrganizationID: organizationID,
		ID:             id,
		Active:         active,
		UserName:       input.UserName,
		DisplayName:    nullableString(input.DisplayName),
	})
	if err != nil {
		return UserRecord{}, err
	}
	return userRecord(row.ID, row.OrganizationID, row.UserID, row.ExternalID, row.UserName, row.DisplayName, row.Active, row.CreatedAt, row.UpdatedAt), nil
}

func (r *Repository) DeleteUser(ctx context.Context, organizationID, id uuid.UUID) error {
	_, err := r.queries.DeleteSCIMUser(ctx, sqlc.DeleteSCIMUserParams{
		OrganizationID: organizationID,
		ID:             id,
	})
	return err
}

func (r *Repository) CreateGroup(ctx context.Context, organizationID, id uuid.UUID, input GroupInput) (GroupRecord, error) {
	row, err := r.queries.CreateSCIMGroup(ctx, sqlc.CreateSCIMGroupParams{
		ID:             id,
		OrganizationID: organizationID,
		ExternalID:     nullableString(input.ExternalID),
		DisplayName:    input.DisplayName,
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return groupRecord(row), nil
}

func (r *Repository) GetGroup(ctx context.Context, organizationID, id uuid.UUID) (GroupRecord, error) {
	row, err := r.queries.GetSCIMGroup(ctx, sqlc.GetSCIMGroupParams{
		OrganizationID: organizationID,
		ID:             id,
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return groupRecord(row), nil
}

func (r *Repository) GetGroupByDisplayName(ctx context.Context, organizationID uuid.UUID, displayName string) (GroupRecord, error) {
	row, err := r.queries.GetSCIMGroupByDisplayName(ctx, sqlc.GetSCIMGroupByDisplayNameParams{
		OrganizationID: organizationID,
		DisplayName:    displayName,
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return groupRecord(row), nil
}

func (r *Repository) ListGroups(ctx context.Context, organizationID uuid.UUID, limit, offset int32) ([]GroupRecord, int64, error) {
	rows, err := r.queries.ListSCIMGroups(ctx, sqlc.ListSCIMGroupsParams{
		OrganizationID: organizationID,
		LimitCount:     limit,
		OffsetCount:    offset,
	})
	if err != nil {
		return nil, 0, err
	}
	total, err := r.queries.CountSCIMGroups(ctx, organizationID)
	if err != nil {
		return nil, 0, err
	}
	out := make([]GroupRecord, 0, len(rows))
	for _, row := range rows {
		out = append(out, groupRecord(row))
	}
	return out, total, nil
}

func (r *Repository) ReplaceGroup(ctx context.Context, organizationID, id uuid.UUID, input GroupInput) (GroupRecord, error) {
	row, err := r.queries.ReplaceSCIMGroup(ctx, sqlc.ReplaceSCIMGroupParams{
		ExternalID:     nullableString(input.ExternalID),
		DisplayName:    input.DisplayName,
		OrganizationID: organizationID,
		ID:             id,
	})
	if err != nil {
		return GroupRecord{}, err
	}
	return groupRecord(row), nil
}

func (r *Repository) DeleteGroup(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteSCIMGroup(ctx, sqlc.DeleteSCIMGroupParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) ListGroupMembers(ctx context.Context, organizationID, groupID uuid.UUID) ([]GroupMember, error) {
	rows, err := r.queries.ListSCIMGroupMembers(ctx, sqlc.ListSCIMGroupMembersParams{
		OrganizationID: organizationID,
		GroupID:        groupID,
	})
	if err != nil {
		return nil, err
	}
	out := make([]GroupMember, 0, len(rows))
	for _, row := range rows {
		display := row.UserName
		if row.DisplayName != nil {
			display = *row.DisplayName
		}
		out = append(out, GroupMember{Value: row.ID.String(), Display: display})
	}
	return out, nil
}

func (r *Repository) ReplaceGroupMembers(ctx context.Context, organizationID, groupID uuid.UUID, memberIDs []uuid.UUID) ([]uuid.UUID, error) {
	return r.queries.ReplaceSCIMGroupMembers(ctx, sqlc.ReplaceSCIMGroupMembersParams{
		OrganizationID: organizationID,
		GroupID:        groupID,
		MemberIds:      memberIDs,
	})
}

func timestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
	}
}

func nullableString(value string) *string {
	if value == "" {
		return nil
	}
	return &value
}

func userRecord(id, organizationID, userID uuid.UUID, externalID, userName string, displayName *string, active bool, createdAt, updatedAt pgtype.Timestamptz) UserRecord {
	return UserRecord{
		ID: id, OrganizationID: organizationID, UserID: userID,
		ExternalID: externalID, UserName: userName, DisplayName: displayName, Active: active,
		CreatedAt: pgconv.TimestamptzToTime(createdAt), UpdatedAt: pgconv.TimestamptzToTime(updatedAt),
	}
}

func groupRecord(row sqlc.ScimGroup) GroupRecord {
	return GroupRecord{
		ID: row.ID, OrganizationID: row.OrganizationID,
		ExternalID: row.ExternalID, DisplayName: row.DisplayName,
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

func tokenFromCreate(row sqlc.CreateSCIMTokenRow) Token {
	return Token{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Prefix:         row.TokenPrefix,
		ExpiresAt:      pgconv.TimestamptzToTimePtr(row.ExpiresAt),
		LastUsedAt:     pgconv.TimestamptzToTimePtr(row.LastUsedAt),
		RevokedAt:      pgconv.TimestamptzToTimePtr(row.RevokedAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
func tokenFromList(row sqlc.ListSCIMTokensRow) Token {
	return Token{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Prefix:         row.TokenPrefix,
		ExpiresAt:      pgconv.TimestamptzToTimePtr(row.ExpiresAt),
		LastUsedAt:     pgconv.TimestamptzToTimePtr(row.LastUsedAt),
		RevokedAt:      pgconv.TimestamptzToTimePtr(row.RevokedAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}
func tokenFromRevoke(row sqlc.RevokeSCIMTokenRow) Token {
	return Token{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Prefix:         row.TokenPrefix,
		ExpiresAt:      pgconv.TimestamptzToTimePtr(row.ExpiresAt),
		LastUsedAt:     pgconv.TimestamptzToTimePtr(row.LastUsedAt),
		RevokedAt:      pgconv.TimestamptzToTimePtr(row.RevokedAt),
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

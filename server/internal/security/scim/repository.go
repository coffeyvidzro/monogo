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
func timestamp(value *time.Time) pgtype.Timestamptz {
	if value == nil {
		return pgtype.Timestamptz{}
	}
	return pgtype.Timestamptz{
		Time:  value.UTC(),
		Valid: true,
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

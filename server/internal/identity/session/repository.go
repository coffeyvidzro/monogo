package session

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) Create(
	ctx context.Context,
	arg sqlc.CreateSessionParams,
) (sqlc.Session, error) {
	return r.queries.CreateSession(ctx, arg)
}

func (r *Repository) GetByTokenHash(
	ctx context.Context,
	tokenHash string,
) (sqlc.Session, error) {
	return r.queries.GetSessionByTokenHash(ctx, tokenHash)
}

func (r *Repository) ListByUserID(
	ctx context.Context,
	userID uuid.UUID,
) ([]sqlc.Session, error) {
	return r.queries.ListSessionsByUserID(ctx, userID)
}

func (r *Repository) Revoke(
	ctx context.Context,
	arg sqlc.RevokeSessionParams,
) error {
	return r.queries.RevokeSession(ctx, arg)
}

func (r *Repository) RevokeUserSessions(
	ctx context.Context,
	userID uuid.UUID,
) error {
	return r.queries.RevokeUserSessions(ctx, userID)
}

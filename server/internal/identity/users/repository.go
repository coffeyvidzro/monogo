package users

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.User, error) {
	return r.queries.GetUserByID(ctx, id)
}

func (r *Repository) UpdateProfile(ctx context.Context, arg sqlc.UpdateUserProfileParams) (sqlc.User, error) {
	return r.queries.UpdateUserProfile(ctx, arg)
}

func (r *Repository) Disable(ctx context.Context, id uuid.UUID) error {
	return r.queries.DisableUser(ctx, id)
}

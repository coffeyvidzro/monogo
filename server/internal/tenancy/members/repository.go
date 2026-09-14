package members

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

func (r *Repository) Add(ctx context.Context, arg sqlc.AddOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	return r.queries.AddOrganizationMember(ctx, arg)
}

func (r *Repository) Get(ctx context.Context, arg sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	return r.queries.GetOrganizationMember(ctx, arg)
}

func (r *Repository) ListByOrganizationID(ctx context.Context, organizationID uuid.UUID) ([]sqlc.OrganizationMember, error) {
	return r.queries.ListMembersByOrganizationID(ctx, organizationID)
}

func (r *Repository) UpdateRole(ctx context.Context, arg sqlc.UpdateMemberRoleParams) (sqlc.OrganizationMember, error) {
	return r.queries.UpdateMemberRole(ctx, arg)
}

func (r *Repository) Disable(ctx context.Context, arg sqlc.DisableOrganizationMemberParams) error {
	return r.queries.DisableOrganizationMember(ctx, arg)
}

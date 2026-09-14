package organization

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

func (r *Repository) Create(ctx context.Context, name string) (sqlc.Organization, error) {
	return r.queries.CreateOrganization(ctx, name)
}

func (r *Repository) CreateWithOwner(ctx context.Context, name string, userID uuid.UUID) (sqlc.Organization, error) {
	row, err := r.queries.CreateOrganizationWithOwner(ctx, sqlc.CreateOrganizationWithOwnerParams{
		Name:   name,
		UserID: userID,
	})
	if err != nil {
		return sqlc.Organization{}, err
	}
	return sqlc.Organization(row), nil
}

func (r *Repository) AddMember(ctx context.Context, arg sqlc.AddOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	return r.queries.AddOrganizationMember(ctx, arg)
}

func (r *Repository) GetByID(ctx context.Context, id uuid.UUID) (sqlc.Organization, error) {
	return r.queries.GetOrganizationByID(ctx, id)
}

func (r *Repository) GetMember(ctx context.Context, arg sqlc.GetOrganizationMemberParams) (sqlc.OrganizationMember, error) {
	return r.queries.GetOrganizationMember(ctx, arg)
}

func (r *Repository) Update(ctx context.Context, arg sqlc.UpdateOrganizationParams) (sqlc.Organization, error) {
	return r.queries.UpdateOrganization(ctx, arg)
}

func (r *Repository) Delete(ctx context.Context, id uuid.UUID) error {
	return r.queries.DeleteOrganization(ctx, id)
}

func (r *Repository) ListByUserID(ctx context.Context, userID uuid.UUID) ([]sqlc.ListOrganizationsByUserIDRow, error) {
	return r.queries.ListOrganizationsByUserID(ctx, userID)
}

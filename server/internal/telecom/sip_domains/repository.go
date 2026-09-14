package sip_domains

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

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	domain string,
) (sqlc.SipDomain, error) {
	return r.queries.CreateSipDomain(ctx, sqlc.CreateSipDomainParams{
		OrganizationID: organizationID,
		Domain:         domain,
	})
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.SipDomain, error) {
	return r.queries.ListSipDomainsByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.SipDomain, error) {
	return r.queries.GetSipDomainByID(ctx, sqlc.GetSipDomainByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	domain *string,
) (sqlc.SipDomain, error) {
	return r.queries.UpdateSipDomain(ctx, sqlc.UpdateSipDomainParams{
		ID:             id,
		OrganizationID: organizationID,
		Domain:         domain,
	})
}

func (r *Repository) Disable(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	return r.queries.DisableSipDomain(ctx, sqlc.DisableSipDomainParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

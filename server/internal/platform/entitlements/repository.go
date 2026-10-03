package entitlements

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]Entitlement, error) {
	rows, err := r.queries.ListEntitlements(ctx, organizationID)
	if err != nil {
		return nil, err
	}

	result := make([]Entitlement, 0, len(rows))
	for _, row := range rows {
		result = append(result, entitlementFromRow(row))
	}
	return result, nil
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	capability Capability,
) (Entitlement, error) {
	row, err := r.queries.GetEntitlement(
		ctx,
		sqlc.GetEntitlementParams{
			OrganizationID: organizationID,
			Capability:     string(capability),
		},
	)
	if err != nil {
		return Entitlement{}, err
	}
	return entitlementFromRow(row), nil
}

func (r *Repository) Set(
	ctx context.Context,
	organizationID uuid.UUID,
	capability Capability,
	enabled bool,
) (Entitlement, error) {
	row, err := r.queries.UpsertEntitlement(
		ctx,
		sqlc.UpsertEntitlementParams{
			OrganizationID: organizationID,
			Capability:     string(capability),
			Enabled:        enabled,
		},
	)
	if err != nil {
		return Entitlement{}, err
	}
	return entitlementFromRow(row), nil
}

func entitlementFromRow(row sqlc.Entitlement) Entitlement {
	return Entitlement{
		OrganizationID: row.OrganizationID,
		Capability:     Capability(row.Capability),
		Enabled:        row.Enabled,
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

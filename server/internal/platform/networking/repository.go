package networking

import (
	"context"
	"net/netip"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{
		queries: queries,
	}
}
func (r *Repository) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest, prefix netip.Prefix) (Policy, error) {
	row, err := r.queries.CreateNetworkPolicy(ctx, sqlc.CreateNetworkPolicyParams{
		ID:             uuid.New(),
		OrganizationID: organizationID,
		Name:           req.Name,
		Action:         req.Action,
		SourceCidr:     prefix,
	})
	if err != nil {
		return Policy{}, err
	}
	return fromRow(row), nil
}
func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]Policy, error) {
	rows, err := r.queries.ListNetworkPolicies(ctx, organizationID)
	if err != nil {
		return nil, err
	}
	out := make([]Policy, 0, len(rows))
	for _, row := range rows {
		out = append(out, fromRow(row))
	}
	return out, nil
}
func (r *Repository) Update(ctx context.Context, organizationID, id uuid.UUID, req UpdateRequest, prefix *netip.Prefix) (Policy, error) {
	row, err := r.queries.UpdateNetworkPolicy(ctx, sqlc.UpdateNetworkPolicyParams{
		Name:           req.Name,
		Action:         req.Action,
		SourceCidr:     prefix,
		Status:         req.Status,
		ID:             id,
		OrganizationID: organizationID,
	})
	if err != nil {
		return Policy{}, err
	}
	return fromRow(row), nil
}
func (r *Repository) Delete(ctx context.Context, organizationID, id uuid.UUID) error {
	return r.queries.DeleteNetworkPolicy(ctx, sqlc.DeleteNetworkPolicyParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}
func fromRow(row sqlc.NetworkPolicy) Policy {
	return Policy{
		ID:             row.ID,
		OrganizationID: row.OrganizationID,
		Name:           row.Name,
		Action:         row.Action,
		SourceCIDR:     row.SourceCidr,
		Status:         row.Status,
		CreatedAt:      pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

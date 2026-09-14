package trunks

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return NewRepository(r.queries.WithTx(tx))
}

func (r *Repository) Create(ctx context.Context, arg sqlc.CreateTrunkParams) (sqlc.Trunk, error) {
	return r.queries.CreateTrunk(ctx, arg)
}

func (r *Repository) CreateManaged(ctx context.Context, organizationID uuid.UUID, name string, direction, status *string) (sqlc.Trunk, error) {
	return r.queries.CreateManagedTrunk(ctx, sqlc.CreateManagedTrunkParams{
		OrganizationID: &organizationID,
		Name:           name,
		Direction:      direction,
		Status:         status,
	})
}

func (r *Repository) CreateCredential(ctx context.Context, organizationID, trunkID uuid.UUID, username, realm, ha1 string) (sqlc.TrunkCredential, error) {
	return r.queries.CreateTrunkCredential(ctx, sqlc.CreateTrunkCredentialParams{
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
		Username:       username,
		Realm:          realm,
		Ha1Md5:         ha1,
	})
}

func (r *Repository) RotateCredential(ctx context.Context, organizationID, trunkID uuid.UUID, username, realm, ha1 string) (sqlc.TrunkCredential, error) {
	return r.queries.RotateTrunkCredential(ctx, sqlc.RotateTrunkCredentialParams{
		TrunkID:        trunkID,
		OrganizationID: organizationID,
		Username:       username,
		Realm:          realm,
		Ha1Md5:         ha1,
	})
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.Trunk, error) {
	return r.queries.ListTrunksByOrganizationID(ctx, &organizationID)
}

func (r *Repository) Get(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (sqlc.Trunk, error) {
	return r.queries.GetTrunkByID(ctx, sqlc.GetTrunkByIDParams{
		ID:             id,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) Update(ctx context.Context, arg sqlc.UpdateTrunkParams) (sqlc.Trunk, error) {
	return r.queries.UpdateTrunk(ctx, arg)
}

func (r *Repository) Disable(ctx context.Context, organizationID uuid.UUID, id uuid.UUID) (sqlc.Trunk, error) {
	return r.queries.DisableTrunk(ctx, sqlc.DisableTrunkParams{
		ID:             id,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) CreateEndpoint(ctx context.Context, arg sqlc.CreateTrunkEndpointParams) (sqlc.TrunkEndpoint, error) {
	return r.queries.CreateTrunkEndpoint(ctx, arg)
}

func (r *Repository) ListEndpoints(ctx context.Context, organizationID uuid.UUID, trunkID uuid.UUID) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListTrunkEndpoints(ctx, sqlc.ListTrunkEndpointsParams{
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) GetEndpoint(ctx context.Context, organizationID uuid.UUID, trunkID uuid.UUID, id uuid.UUID) (sqlc.TrunkEndpoint, error) {
	return r.queries.GetTrunkEndpointByID(ctx, sqlc.GetTrunkEndpointByIDParams{
		ID:             id,
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) UpdateEndpoint(ctx context.Context, arg sqlc.UpdateTrunkEndpointParams) (sqlc.TrunkEndpoint, error) {
	return r.queries.UpdateTrunkEndpoint(ctx, arg)
}

func (r *Repository) DeleteEndpoint(ctx context.Context, organizationID uuid.UUID, trunkID uuid.UUID, id uuid.UUID) (sqlc.TrunkEndpoint, error) {
	return r.queries.DeleteTrunkEndpoint(ctx, sqlc.DeleteTrunkEndpointParams{
		ID:             id,
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
	})
}

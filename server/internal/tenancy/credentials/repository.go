package credentials

import (
	"context"
	"encoding/json"
	"net/netip"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Create(ctx context.Context, input CreateInput, tokenHash, tokenPrefix string) (sqlc.OrganizationToken, error) {
	scopes, err := json.Marshal(input.Scopes)
	if err != nil {
		return sqlc.OrganizationToken{}, err
	}

	return r.queries.CreateOrganizationToken(ctx, sqlc.CreateOrganizationTokenParams{
		OrganizationID: input.OrganizationID,
		CreatedBy:      &input.CreatedBy,
		Name:           input.Name,
		Description:    input.Description,
		TokenHash:      tokenHash,
		TokenPrefix:    tokenPrefix,
		Scopes:         scopes,
		ExpiresAt:      pgconv.NullableTimestamptz(input.ExpiresAt),
	})
}

func (r *Repository) GetByID(ctx context.Context, organizationID, id uuid.UUID) (sqlc.OrganizationToken, error) {
	return r.queries.GetOrganizationTokenByID(ctx, sqlc.GetOrganizationTokenByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) GetByTokenHash(ctx context.Context, tokenHash string) (sqlc.OrganizationToken, error) {
	return r.queries.GetOrganizationTokenByTokenHash(ctx, tokenHash)
}

func (r *Repository) List(ctx context.Context, organizationID uuid.UUID) ([]sqlc.ListOrganizationTokensByOrganizationIDRow, error) {
	return r.queries.ListOrganizationTokensByOrganizationID(ctx, organizationID)
}

func (r *Repository) Update(ctx context.Context, input UpdateInput, actorID uuid.UUID) (sqlc.OrganizationToken, error) {
	var scopes []byte
	if input.Scopes != nil {
		var err error
		scopes, err = json.Marshal(*input.Scopes)
		if err != nil {
			return sqlc.OrganizationToken{}, err
		}
	}

	return r.queries.UpdateOrganizationToken(ctx, sqlc.UpdateOrganizationTokenParams{
		ID:             input.ID,
		OrganizationID: input.OrganizationID,
		ActorUserID:    actorID,
		Name:           input.Name,
		Description:    input.Description,
		Scopes:         scopes,
		ExpiresAt:      pgconv.NullableTimestamptz(input.ExpiresAt),
	})
}

func (r *Repository) Disable(ctx context.Context, organizationID, id, actorID uuid.UUID) error {
	return r.queries.DisableOrganizationToken(ctx, sqlc.DisableOrganizationTokenParams{
		ID:             id,
		OrganizationID: organizationID,
		ActorUserID:    actorID,
	})
}

func (r *Repository) Touch(ctx context.Context, id uuid.UUID, ip netip.Addr) error {
	return r.queries.TouchOrganizationToken(ctx, sqlc.TouchOrganizationTokenParams{
		ID:         id,
		LastUsedIp: &ip,
	})
}

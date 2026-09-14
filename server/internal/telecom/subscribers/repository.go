package subscribers

import (
	"context"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/pkg/hasher"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) Domain(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.SipDomain, error) {
	return r.queries.GetSipDomainByID(ctx, sqlc.GetSipDomainByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
	hashes hasher.HA1Hashes,
) (sqlc.Subscriber, error) {
	return r.queries.CreateSubscriber(ctx, sqlc.CreateSubscriberParams{
		OrganizationID: organizationID,
		SipDomainID:    req.SIPDomainID,
		Username:       req.Username,
		Ha1Md5:         &hashes.MD5,
		Ha1Sha256:      &hashes.SHA256,
		Ha1Sha512256:   &hashes.SHA512_256,
		DisplayName:    req.DisplayName,
	})
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Subscriber, error) {
	return r.queries.ListSubscribersByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.Subscriber, error) {
	return r.queries.GetSubscriberByID(ctx, sqlc.GetSubscriberByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Update(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	name *string,
) (sqlc.Subscriber, error) {
	return r.queries.UpdateSubscriber(ctx, sqlc.UpdateSubscriberParams{
		ID:             id,
		OrganizationID: organizationID,
		DisplayName:    name,
	})
}

func (r *Repository) Rotate(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
	hashes hasher.HA1Hashes,
) (sqlc.Subscriber, error) {
	return r.queries.SetSubscriberPassword(ctx, sqlc.SetSubscriberPasswordParams{
		ID:             id,
		OrganizationID: organizationID,
		Ha1Md5:         &hashes.MD5,
		Ha1Sha256:      &hashes.SHA256,
		Ha1Sha512256:   &hashes.SHA512_256,
	})
}

func (r *Repository) Disable(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	return r.queries.DisableSubscriber(ctx, sqlc.DisableSubscriberParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

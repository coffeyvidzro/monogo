package carriers

import (
	"context"
	"net/netip"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) WithTx(tx pgx.Tx) *Repository {
	return &Repository{queries: r.queries.WithTx(tx)}
}

func (r *Repository) Create(ctx context.Context, p sqlc.CreateCarrierConnectionParams) (sqlc.CarrierConnection, error) {
	return r.queries.CreateCarrierConnection(ctx, p)
}

func (r *Repository) Get(ctx context.Context, org, id uuid.UUID) (sqlc.GetCarrierConnectionByIDRow, error) {
	return r.queries.GetCarrierConnectionByID(ctx, sqlc.GetCarrierConnectionByIDParams{ID: id, OrganizationID: &org})
}

func (r *Repository) List(
	ctx context.Context,
	org uuid.UUID,
) ([]sqlc.ListCarrierConnectionsByOrganizationIDRow, error) {
	return r.queries.ListCarrierConnectionsByOrganizationID(ctx, &org)
}

func (r *Repository) Update(ctx context.Context, org, id uuid.UUID, req UpdateRequest) (sqlc.CarrierConnection, error) {
	var codecs []string
	if req.Codecs != nil {
		codecs = *req.Codecs
	}
	return r.queries.UpdateCarrierConnection(ctx, sqlc.UpdateCarrierConnectionParams{
		Name:               req.Name,
		Status:             req.Status,
		InboundEnabled:     req.InboundEnabled,
		MaxCps:             req.MaxCPS,
		MaxConcurrentCalls: req.MaxConcurrentCalls,
		MaxDailyMinutes:    req.MaxDailyMinutes,
		Codecs:             codecs,
		SupportsVideo:      req.SupportsVideo,
		SupportsFax:        req.SupportsFax,
		ID:                 id,
		OrganizationID:     &org,
	})
}

func (r *Repository) Disable(ctx context.Context, org, id uuid.UUID) error {
	return r.queries.DisableCarrierConnection(ctx, sqlc.DisableCarrierConnectionParams{ID: id, OrganizationID: &org})
}

func (r *Repository) SetOutboundDigest(ctx context.Context, org, id uuid.UUID, username, realm, ciphertext, ha1 string) error {
	return r.queries.SetCarrierConnectionOutboundDigestAuth(ctx, sqlc.SetCarrierConnectionOutboundDigestAuthParams{
		AuthUsername:         &username,
		AuthSecretCiphertext: &ciphertext,
		AuthRealm:            &realm,
		AuthHa1Md5:           &ha1,
		ID:                   id,
		OrganizationID:       &org,
	})
}

func (r *Repository) ClearOutbound(ctx context.Context, org, id uuid.UUID) error {
	return r.queries.ClearCarrierConnectionOutboundAuth(ctx, sqlc.ClearCarrierConnectionOutboundAuthParams{
		ID:             id,
		OrganizationID: &org,
	})
}

func (r *Repository) SetInboundDigest(ctx context.Context, org, id uuid.UUID, username, realm, ciphertext, ha1 string) error {
	return r.queries.SetCarrierConnectionInboundDigestAuth(ctx, sqlc.SetCarrierConnectionInboundDigestAuthParams{
		InboundUsername:         &username,
		InboundSecretCiphertext: &ciphertext,
		InboundRealm:            &realm,
		InboundHa1Md5:           &ha1,
		ID:                      id,
		OrganizationID:          &org,
	})
}

func (r *Repository) SetInboundIP(ctx context.Context, org, id uuid.UUID) error {
	return r.queries.SetCarrierConnectionInboundIPAuth(ctx, sqlc.SetCarrierConnectionInboundIPAuthParams{
		ID:             id,
		OrganizationID: &org,
	})
}

func (r *Repository) SetInboundNone(ctx context.Context, org, id uuid.UUID) error {
	return r.queries.SetCarrierConnectionInboundNoAuth(ctx, sqlc.SetCarrierConnectionInboundNoAuthParams{
		ID:             id,
		OrganizationID: &org,
	})
}

func (r *Repository) CreateSourceIP(
	ctx context.Context,
	org, id uuid.UUID,
	cidr netip.Prefix,
) (sqlc.CarrierConnectionSourceIp, error) {
	return r.queries.CreateCarrierConnectionSourceIP(ctx, sqlc.CreateCarrierConnectionSourceIPParams{
		OrganizationID:      &org,
		CarrierConnectionID: id,
		Cidr:                cidr,
	})
}

func (r *Repository) ListSourceIPs(ctx context.Context, org, id uuid.UUID) ([]sqlc.CarrierConnectionSourceIp, error) {
	return r.queries.ListCarrierConnectionSourceIPs(ctx, sqlc.ListCarrierConnectionSourceIPsParams{
		CarrierConnectionID: id,
		OrganizationID:      &org,
	})
}

func (r *Repository) DeleteSourceIP(ctx context.Context, org, id, sourceID uuid.UUID) error {
	return r.queries.DeleteCarrierConnectionSourceIP(ctx, sqlc.DeleteCarrierConnectionSourceIPParams{
		ID:                  sourceID,
		CarrierConnectionID: id,
		OrganizationID:      &org,
	})
}

func (r *Repository) ListTrunks(ctx context.Context, org, id uuid.UUID) ([]sqlc.Trunk, error) {
	return r.queries.ListTrunksByCarrierConnectionID(ctx, sqlc.ListTrunksByCarrierConnectionIDParams{
		CarrierConnectionID: &id, OrganizationID: &org,
	})
}

func (r *Repository) ListTrunkEndpoints(ctx context.Context, org, trunkID uuid.UUID) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListTrunkEndpoints(ctx, sqlc.ListTrunkEndpointsParams{TrunkID: trunkID, OrganizationID: &org})
}

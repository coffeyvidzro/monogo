package trunks

import (
	"context"
	"net/netip"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
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

func (r *Repository) Create(
	ctx context.Context,
	arg sqlc.CreateTrunkParams,
) (sqlc.Trunk, error) {
	return r.queries.CreateTrunk(ctx, arg)
}

func (r *Repository) InsertDigest(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
	direction, username, realm, ha1 string,
) error {
	return r.queries.InsertTrunkDigestCredential(
		ctx,
		sqlc.InsertTrunkDigestCredentialParams{
			TrunkID:        trunkID,
			OrganizationID: organizationID,
			Direction:      direction,
			Username:       username,
			Realm:          realm,
			Ha1Md5:         ha1,
		},
	)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.Trunk, error) {
	return r.queries.ListTrunksByOrganizationID(ctx, organizationID)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Trunk, error) {
	return r.queries.GetTrunkByID(ctx, sqlc.GetTrunkByIDParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) Update(
	ctx context.Context,
	arg sqlc.UpdateTrunkParams,
) (sqlc.Trunk, error) {
	return r.queries.UpdateTrunk(ctx, arg)
}

func (r *Repository) Disable(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Trunk, error) {
	return r.queries.DisableTrunk(ctx, sqlc.DisableTrunkParams{
		ID:             id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) SetOutboundDigest(
	ctx context.Context,
	organizationID, id uuid.UUID,
	username, realm, ciphertext, ha1 string,
) error {
	return r.queries.SetTrunkOutboundDigestAuth(
		ctx,
		sqlc.SetTrunkOutboundDigestAuthParams{
			AuthUsername:         &username,
			AuthRealm:            &realm,
			AuthSecretCiphertext: &ciphertext,
			AuthHa1Md5:           &ha1,
			ID:                   id,
			OrganizationID:       organizationID,
		},
	)
}

func (r *Repository) ClearOutbound(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	return r.queries.ClearTrunkOutboundAuth(
		ctx,
		sqlc.ClearTrunkOutboundAuthParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) SetInboundDigest(
	ctx context.Context,
	organizationID, id uuid.UUID,
	username, realm, ciphertext, ha1 string,
) error {
	return r.queries.SetTrunkInboundDigestAuth(
		ctx,
		sqlc.SetTrunkInboundDigestAuthParams{
			InboundUsername:         &username,
			InboundRealm:            &realm,
			InboundSecretCiphertext: &ciphertext,
			InboundHa1Md5:           &ha1,
			ID:                      id,
			OrganizationID:          organizationID,
		},
	)
}

func (r *Repository) SetInboundIP(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	return r.queries.SetTrunkInboundIPAuth(
		ctx,
		sqlc.SetTrunkInboundIPAuthParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) SetInboundNone(
	ctx context.Context,
	organizationID, id uuid.UUID,
) error {
	return r.queries.SetTrunkInboundNoAuth(
		ctx,
		sqlc.SetTrunkInboundNoAuthParams{
			ID:             id,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) CreateSourceIP(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
	cidr netip.Prefix,
) (sqlc.TrunkSourceIp, error) {
	return r.queries.CreateTrunkSourceIP(
		ctx,
		sqlc.CreateTrunkSourceIPParams{
			OrganizationID: organizationID,
			TrunkID:        trunkID,
			Cidr:           cidr,
		},
	)
}

func (r *Repository) ListSourceIPs(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
) ([]sqlc.TrunkSourceIp, error) {
	return r.queries.ListTrunkSourceIPs(
		ctx,
		sqlc.ListTrunkSourceIPsParams{
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) DeleteSourceIP(
	ctx context.Context,
	organizationID, trunkID, sourceIPID uuid.UUID,
) error {
	return r.queries.DeleteTrunkSourceIP(
		ctx,
		sqlc.DeleteTrunkSourceIPParams{
			ID:             sourceIPID,
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) CreateEndpoint(
	ctx context.Context,
	arg sqlc.CreateTrunkEndpointParams,
) (sqlc.TrunkEndpoint, error) {
	return r.queries.CreateTrunkEndpoint(ctx, arg)
}

func (r *Repository) ListEndpoints(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListTrunkEndpoints(
		ctx,
		sqlc.ListTrunkEndpointsParams{
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) GetEndpoint(
	ctx context.Context,
	organizationID, trunkID, id uuid.UUID,
) (sqlc.TrunkEndpoint, error) {
	return r.queries.GetTrunkEndpointByID(
		ctx,
		sqlc.GetTrunkEndpointByIDParams{
			ID:             id,
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) UpdateEndpoint(
	ctx context.Context,
	arg sqlc.UpdateTrunkEndpointParams,
) (sqlc.TrunkEndpoint, error) {
	return r.queries.UpdateTrunkEndpoint(ctx, arg)
}

func (r *Repository) DeleteEndpoint(
	ctx context.Context,
	organizationID, trunkID, id uuid.UUID,
) (sqlc.TrunkEndpoint, error) {
	return r.queries.DeleteTrunkEndpoint(
		ctx,
		sqlc.DeleteTrunkEndpointParams{
			ID:             id,
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
}

func (r *Repository) ListForHealthCheck(
	ctx context.Context,
	checkedAt, dueBefore time.Time,
	batchSize int32,
) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListTrunkEndpointsForHealthCheck(
		ctx,
		sqlc.ListTrunkEndpointsForHealthCheckParams{
			CheckedAt: pgconv.TimeToTimestamptz(checkedAt),
			DueBefore: pgconv.TimeToTimestamptz(dueBefore),
			BatchSize: batchSize,
		},
	)
}

func (r *Repository) MarkHealthy(
	ctx context.Context,
	id uuid.UUID,
	checkedAt time.Time,
	responseCode, latencyMs int32,
) error {
	_, err := r.queries.MarkTrunkEndpointHealthy(
		ctx,
		sqlc.MarkTrunkEndpointHealthyParams{
			CheckedAt:    pgconv.TimeToTimestamptz(checkedAt),
			ResponseCode: &responseCode,
			LatencyMs:    &latencyMs,
			ID:           id,
		},
	)
	return err
}

func (r *Repository) MarkProbeFailed(
	ctx context.Context,
	id uuid.UUID,
	checkedAt time.Time,
	latencyMs int32,
	message string,
	threshold int32,
	cooldownUntil time.Time,
) error {
	_, err := r.queries.MarkTrunkEndpointProbeFailed(
		ctx,
		sqlc.MarkTrunkEndpointProbeFailedParams{
			FailureThreshold: threshold,
			CheckedAt:        pgconv.TimeToTimestamptz(checkedAt),
			LatencyMs:        &latencyMs,
			LastError:        &message,
			CooldownUntil:    pgconv.TimeToTimestamptz(cooldownUntil),
			ID:               id,
		},
	)
	return err
}

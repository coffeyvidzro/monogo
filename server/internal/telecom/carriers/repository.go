package carriers

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/modules/audit"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

func NewRepository(db *pgxpool.Pool) *Repository {
	return &Repository{db: db, queries: sqlc.New(db)}
}

func (r *Repository) Create(
	ctx context.Context,
	arg sqlc.CreateCarrierConnectionParams,
) (sqlc.CarrierConnection, error) {
	return r.queries.CreateCarrierConnection(ctx, arg)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
) ([]sqlc.ListCarrierConnectionsByOrganizationIDRow, error) {
	return r.queries.ListCarrierConnectionsByOrganizationID(ctx, &organizationID)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) (sqlc.GetCarrierConnectionByIDRow, error) {
	return r.queries.GetCarrierConnectionByID(
		ctx,
		sqlc.GetCarrierConnectionByIDParams{
			ID:             id,
			OrganizationID: &organizationID,
		},
	)
}

func (r *Repository) Update(
	ctx context.Context,
	arg sqlc.UpdateCarrierConnectionParams,
) (sqlc.CarrierConnection, error) {
	return r.queries.UpdateCarrierConnection(ctx, arg)
}

func (r *Repository) Disable(
	ctx context.Context,
	organizationID uuid.UUID,
	id uuid.UUID,
) error {
	return r.queries.DisableCarrierConnection(
		ctx,
		sqlc.DisableCarrierConnectionParams{
			ID:             id,
			OrganizationID: &organizationID,
		},
	)
}

func (r *Repository) SetDigestAuth(ctx context.Context, org, id uuid.UUID, direction, username, realm, ciphertext, ha1 string, event audit.Event) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if direction == "outbound" {
		err = q.SetCarrierConnectionOutboundDigestAuth(ctx, sqlc.SetCarrierConnectionOutboundDigestAuthParams{ID: id, OrganizationID: &org, AuthUsername: &username, AuthSecretCiphertext: &ciphertext})
	} else {
		err = q.SetCarrierConnectionInboundDigestAuth(ctx, sqlc.SetCarrierConnectionInboundDigestAuthParams{ID: id, OrganizationID: &org, InboundUsername: &username, InboundSecretCiphertext: &ciphertext})
	}
	if err != nil {
		return err
	}
	_, err = tx.Exec(ctx, `INSERT INTO carrier_digest_credentials (carrier_connection_id, organization_id, direction, username, realm, ha1_md5) VALUES ($1,$2,$3,$4,$5,$6) ON CONFLICT (carrier_connection_id, direction) DO UPDATE SET username=EXCLUDED.username, realm=EXCLUDED.realm, ha1_md5=EXCLUDED.ha1_md5, updated_at=now()`, id, org, direction, username, realm, ha1)
	if err != nil {
		return err
	}
	if err := audit.Insert(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) ClearAuth(ctx context.Context, org, id uuid.UUID, direction string, event audit.Event) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	q := r.queries.WithTx(tx)
	if direction == "outbound" {
		err = q.ClearCarrierConnectionOutboundAuth(ctx, sqlc.ClearCarrierConnectionOutboundAuthParams{ID: id, OrganizationID: &org})
	} else {
		err = q.SetCarrierConnectionInboundNoAuth(ctx, sqlc.SetCarrierConnectionInboundNoAuthParams{ID: id, OrganizationID: &org})
	}
	if err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM carrier_digest_credentials WHERE carrier_connection_id=$1 AND organization_id=$2 AND direction=$3`, id, org, direction); err != nil {
		return err
	}
	if err := audit.Insert(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) SetInboundIPAuth(ctx context.Context, org, id uuid.UUID, event audit.Event) error {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	if err = r.queries.WithTx(tx).SetCarrierConnectionInboundIPAuth(ctx, sqlc.SetCarrierConnectionInboundIPAuthParams{ID: id, OrganizationID: &org}); err != nil {
		return err
	}
	if _, err = tx.Exec(ctx, `DELETE FROM carrier_digest_credentials WHERE carrier_connection_id=$1 AND organization_id=$2 AND direction='inbound'`, id, org); err != nil {
		return err
	}
	if err := audit.Insert(ctx, tx, event); err != nil {
		return err
	}
	return tx.Commit(ctx)
}

func (r *Repository) CreateSourceIP(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
	cidr netip.Prefix,
) (sqlc.CarrierConnectionSourceIp, error) {
	return r.queries.CreateCarrierConnectionSourceIP(
		ctx,
		sqlc.CreateCarrierConnectionSourceIPParams{
			OrganizationID:      &organizationID,
			CarrierConnectionID: connectionID,
			Cidr:                cidr,
		},
	)
}

func (r *Repository) ListSourceIPs(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
) ([]sqlc.CarrierConnectionSourceIp, error) {
	return r.queries.ListCarrierConnectionSourceIPs(
		ctx,
		sqlc.ListCarrierConnectionSourceIPsParams{
			CarrierConnectionID: connectionID,
			OrganizationID:      &organizationID,
		},
	)
}

func (r *Repository) DeleteSourceIP(
	ctx context.Context,
	organizationID uuid.UUID,
	connectionID uuid.UUID,
	id uuid.UUID,
) error {
	return r.queries.DeleteCarrierConnectionSourceIP(
		ctx,
		sqlc.DeleteCarrierConnectionSourceIPParams{
			ID:                  id,
			CarrierConnectionID: connectionID,
			OrganizationID:      &organizationID,
		},
	)
}

func (r *Repository) ListProviders(ctx context.Context) ([]sqlc.CarrierProvider, error) {
	return r.queries.ListCarrierProviders(ctx)
}

func (r *Repository) GetProvider(ctx context.Context, id uuid.UUID) (sqlc.CarrierProvider, error) {
	return r.queries.GetCarrierProviderByID(ctx, id)
}

func (r *Repository) ListConnectionTrunks(ctx context.Context, organizationID, connectionID uuid.UUID) ([]sqlc.Trunk, error) {
	return r.queries.ListTrunksByCarrierConnectionID(ctx, sqlc.ListTrunksByCarrierConnectionIDParams{
		CarrierConnectionID: &connectionID,
		OrganizationID:      &organizationID,
	})
}

func (r *Repository) ListTrunkEndpoints(ctx context.Context, organizationID, trunkID uuid.UUID) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListTrunkEndpoints(ctx, sqlc.ListTrunkEndpointsParams{
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
	})
}

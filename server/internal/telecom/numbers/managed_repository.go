package numbers

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type ManagedRepository struct{ db *pgxpool.Pool }

func NewManagedRepository(db *pgxpool.Pool) *ManagedRepository { return &ManagedRepository{db: db} }

const managedOrderColumns = `id, organization_id, provider_id, phone_number_id, idempotency_key,
 request_hash, number, country_code, available_did_id, sku_id, provider_order_id, provider_did_id,
 inbound_trunk_id, status, submitted_at, ownership_verified_at, routing_verified_at, activated_at,
 reconcile_after, reconcile_attempts, created_at, updated_at`

type rowScanner interface{ Scan(...any) error }

func scanManagedOrder(row rowScanner) (ManagedOrder, error) {
	var o ManagedOrder
	err := row.Scan(&o.ID, &o.OrganizationID, &o.ProviderID, &o.PhoneNumberID, &o.IdempotencyKey,
		&o.RequestHash, &o.Number, &o.CountryCode, &o.AvailableDIDID, &o.SKUID, &o.ProviderOrderID,
		&o.ProviderDIDID, &o.InboundTrunkID, &o.Status, &o.SubmittedAt, &o.OwnershipVerifiedAt,
		&o.RoutingVerifiedAt, &o.ActivatedAt, &o.ReconcileAfter, &o.ReconciliationAttempts,
		&o.CreatedAt, &o.UpdatedAt)
	return o, err
}

func (r *ManagedRepository) Create(ctx context.Context, organizationID uuid.UUID, key, hash string, request ManagedPurchaseRequest, availableDIDID, skuID string) (ManagedOrder, bool, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return ManagedOrder{}, false, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	id := uuid.New()
	query := `INSERT INTO managed_number_orders (id, organization_id, provider_id, idempotency_key,
 request_hash, number, country_code, available_did_id, sku_id)
 SELECT $1, o.id, cp.id, $2, $3, $4, $5, $6, $7
 FROM organizations o JOIN carrier_providers cp ON cp.slug='didww' AND cp.adapter='didww' AND cp.status='active'
 WHERE o.id=$8 AND o.status='active' AND o.deleted_at IS NULL
 ON CONFLICT (organization_id, idempotency_key) DO NOTHING
 RETURNING ` + managedOrderColumns
	o, err := scanManagedOrder(tx.QueryRow(ctx, query, id, key, hash, request.Number, request.CountryCode, availableDIDID, skuID, organizationID))
	created := err == nil
	if errors.Is(err, pgx.ErrNoRows) {
		o, err = scanManagedOrder(tx.QueryRow(ctx, `SELECT `+managedOrderColumns+` FROM managed_number_orders WHERE organization_id=$1 AND idempotency_key=$2`, organizationID, key))
	}
	if err != nil {
		return ManagedOrder{}, false, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ManagedOrder{}, false, err
	}
	return o, created, nil
}

func (r *ManagedRepository) Get(ctx context.Context, organizationID, id uuid.UUID) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `SELECT `+managedOrderColumns+` FROM managed_number_orders WHERE organization_id=$1 AND id=$2`, organizationID, id))
}
func (r *ManagedRepository) GetInternal(ctx context.Context, id uuid.UUID) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `SELECT `+managedOrderColumns+` FROM managed_number_orders WHERE id=$1`, id))
}

func (r *ManagedRepository) ListDue(ctx context.Context, limit int) ([]ManagedOrder, error) {
	if limit < 1 {
		limit = 50
	}
	rows, err := r.db.Query(ctx, `SELECT `+managedOrderColumns+` FROM managed_number_orders
 WHERE status IN ('submitting','outcome_unknown','provider_pending','configuring')
 AND reconcile_after <= now() ORDER BY reconcile_after, created_at LIMIT $1`, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	orders := make([]ManagedOrder, 0)
	for rows.Next() {
		order, scanErr := scanManagedOrder(rows)
		if scanErr != nil {
			return nil, scanErr
		}
		orders = append(orders, order)
	}
	return orders, rows.Err()
}
func (r *ManagedRepository) ClaimSubmission(ctx context.Context, id uuid.UUID) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET status='submitting', submitted_at=now(), error_code=NULL, error_message=NULL WHERE id=$1 AND status='ready' RETURNING `+managedOrderColumns, id))
}
func (r *ManagedRepository) MarkOutcomeUnknown(ctx context.Context, id uuid.UUID, code, message string, next time.Time) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET status='outcome_unknown', error_code=$2, error_message=$3, reconcile_after=$4 WHERE id=$1 AND status='submitting' RETURNING `+managedOrderColumns, id, code, message, next))
}
func (r *ManagedRepository) RecordProviderOrder(ctx context.Context, id uuid.UUID, providerOrderID string, next time.Time) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET provider_order_id=$2, status='provider_pending', error_code=NULL, error_message=NULL, reconcile_after=$3 WHERE id=$1 AND status IN ('submitting','outcome_unknown','provider_pending') AND (provider_order_id IS NULL OR provider_order_id=$2) RETURNING `+managedOrderColumns, id, providerOrderID, next))
}
func (r *ManagedRepository) RecordOwnedDID(ctx context.Context, id uuid.UUID, didID string, now time.Time) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET provider_did_id=$2, ownership_verified_at=$3, status='configuring', error_code=NULL, error_message=NULL WHERE id=$1 AND status IN ('provider_pending','configuring') AND (provider_did_id IS NULL OR provider_did_id=$2) RETURNING `+managedOrderColumns, id, didID, now))
}
func (r *ManagedRepository) ScheduleReconciliation(ctx context.Context, id uuid.UUID, code string, next time.Time) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET reconcile_after=$2, reconcile_attempts=reconcile_attempts+1, error_code=$3, error_message=NULL WHERE id=$1 AND status IN ('submitting','outcome_unknown','provider_pending','configuring') RETURNING `+managedOrderColumns, id, next, code))
}
func (r *ManagedRepository) ManualReview(ctx context.Context, id uuid.UUID, code string) (ManagedOrder, error) {
	return scanManagedOrder(r.db.QueryRow(ctx, `UPDATE managed_number_orders SET status='manual_review', error_code=$2, error_message='provider identity requires manual review' WHERE id=$1 AND status <> 'completed' RETURNING `+managedOrderColumns, id, code))
}

func (r *ManagedRepository) Activate(ctx context.Context, id uuid.UUID, didID, trunkID string, now time.Time) (ManagedOrder, error) {
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{IsoLevel: pgx.Serializable})
	if err != nil {
		return ManagedOrder{}, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	order, err := scanManagedOrder(tx.QueryRow(ctx, `SELECT `+managedOrderColumns+` FROM managed_number_orders WHERE id=$1 FOR UPDATE`, id))
	if err != nil {
		return ManagedOrder{}, err
	}
	if order.Status == "completed" {
		return order, tx.Commit(ctx)
	}
	if order.Status != "configuring" || order.ProviderDIDID == nil || *order.ProviderDIDID != didID {
		return ManagedOrder{}, pgx.ErrNoRows
	}

	var phoneID uuid.UUID
	err = tx.QueryRow(ctx, `INSERT INTO phone_numbers (organization_id, number, country_code, provisioning_mode,
 provider_id, provider_resource_id, voice_enabled, sms_enabled, status)
 VALUES ($1,$2,$3,'managed',$4,$5,true,false,'active')
 ON CONFLICT (provider_id, provider_resource_id) WHERE provider_id IS NOT NULL AND provider_resource_id IS NOT NULL
 DO UPDATE SET updated_at=phone_numbers.updated_at
 RETURNING id`, order.OrganizationID, order.Number, order.CountryCode, order.ProviderID, didID).Scan(&phoneID)
	if err != nil {
		return ManagedOrder{}, err
	}
	order, err = scanManagedOrder(tx.QueryRow(ctx, `UPDATE managed_number_orders SET phone_number_id=$2,
 inbound_trunk_id=$3, routing_verified_at=$4, activated_at=$4, status='completed', error_code=NULL,
 error_message=NULL WHERE id=$1 AND status='configuring' RETURNING `+managedOrderColumns, id, phoneID, trunkID, now))
	if err != nil {
		return ManagedOrder{}, err
	}
	if err = tx.Commit(ctx); err != nil {
		return ManagedOrder{}, err
	}
	return order, nil
}

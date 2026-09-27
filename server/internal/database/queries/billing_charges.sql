-- name: CreateBillingCharge :one
INSERT INTO charges (
    organization_id,
    wallet_id,
    resource_type,
    resource_id,
    charging_mode,
    currency,
    idempotency_key,
    request_hash,
    pricing_snapshot
)
SELECT
    organization.id,
    wallet.id,
    sqlc.arg(resource_type),
    sqlc.arg(resource_id),
    sqlc.arg(charging_mode),
    wallet.currency,
    sqlc.arg(idempotency_key),
    sqlc.arg(request_hash),
    sqlc.arg(pricing_snapshot)::jsonb
FROM organizations AS organization
JOIN wallets AS wallet
  ON wallet.id = sqlc.arg(wallet_id)
 AND wallet.organization_id = organization.id
WHERE organization.id = sqlc.arg(organization_id)
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
  AND wallet.status = 'active'
  AND wallet.currency = sqlc.arg(currency)
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetBillingCharge :one
SELECT *
FROM charges
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetBillingChargeByIdempotencyKey :one
SELECT *
FROM charges
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: GetBillingChargeByResource :one
SELECT *
FROM charges
WHERE organization_id = sqlc.arg(organization_id)
  AND resource_type = sqlc.arg(resource_type)
  AND resource_id = sqlc.arg(resource_id)
LIMIT 1;

-- name: ListBillingCharges :many
SELECT *
FROM charges
WHERE organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- Usage writes are internal observations, not charges. The database checks
-- that the meter belongs to the organization and is currently active.
-- name: CreateUsageEvent :one
INSERT INTO usage_events (
    organization_id, meter_id, quantity, source_type, source_id,
    idempotency_key, dimensions, occurred_at
)
SELECT m.organization_id, m.id, sqlc.arg(quantity)::BIGINT,
       sqlc.arg(source_type)::TEXT, sqlc.arg(source_id)::TEXT,
       sqlc.arg(idempotency_key)::TEXT, sqlc.arg(dimensions)::JSONB,
       sqlc.arg(occurred_at)::TIMESTAMPTZ
FROM meters AS m
JOIN organizations AS o ON o.id = m.organization_id
WHERE m.id = sqlc.arg(meter_id)::UUID
  AND m.organization_id = sqlc.arg(organization_id)::UUID
  AND m.active = true
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetUsageEventByKey :one
SELECT * FROM usage_events
WHERE organization_id = sqlc.arg(organization_id)::UUID
  AND idempotency_key = sqlc.arg(idempotency_key)::TEXT
LIMIT 1;

-- A repeated event is equivalent only if every material field matches.
-- PostgreSQL JSONB equality avoids differences in key ordering and whitespace.
-- name: GetMatchingUsageEventByKey :one
SELECT * FROM usage_events
WHERE organization_id = sqlc.arg(organization_id)::UUID
  AND idempotency_key = sqlc.arg(idempotency_key)::TEXT
  AND meter_id = sqlc.arg(meter_id)::UUID
  AND quantity = sqlc.arg(quantity)::BIGINT
  AND source_type = sqlc.arg(source_type)::TEXT
  AND source_id = sqlc.arg(source_id)::TEXT
  AND dimensions = sqlc.arg(dimensions)::JSONB
  AND occurred_at = sqlc.arg(occurred_at)::TIMESTAMPTZ
LIMIT 1;

-- name: GetUsageEvent :one
SELECT * FROM usage_events
WHERE organization_id = sqlc.arg(organization_id)::UUID
  AND id = sqlc.arg(id)::UUID
LIMIT 1;

-- name: GetUsageChargeByEvent :one
SELECT uc.* FROM usage_charges AS uc
JOIN usage_events AS ue ON ue.id = uc.usage_event_id
WHERE ue.organization_id = sqlc.arg(organization_id)::UUID
  AND uc.usage_event_id = sqlc.arg(usage_event_id)::UUID
LIMIT 1;

-- name: CreateUsageCharge :one
INSERT INTO usage_charges (
    usage_event_id, wallet_transaction_id, amount_minor, currency
) VALUES (
    sqlc.arg(usage_event_id)::UUID, sqlc.arg(wallet_transaction_id)::UUID,
    sqlc.arg(amount_minor)::BIGINT, sqlc.arg(currency)::TEXT
)
RETURNING *;

-- name: ListUsageEventsByMeter :many
SELECT * FROM usage_events
WHERE organization_id = sqlc.arg(organization_id)::UUID
  AND meter_id = sqlc.arg(meter_id)::UUID
ORDER BY occurred_at DESC, id DESC
LIMIT sqlc.arg(row_limit)::INTEGER;

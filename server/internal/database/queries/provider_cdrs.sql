-- name: CreateProviderCDR :one
INSERT INTO provider_cdrs (
    provider_id,
    carrier_connection_id,
    call_id,
    provider_cdr_id,
    direction,
    source,
    destination,
    started_at,
    answered_at,
    ended_at,
    duration_seconds,
    billable_seconds,
    raw_payload,
    received_at
)
SELECT
    cp.id,
    cc.id,
    c.id,
    sqlc.arg(provider_cdr_id),
    sqlc.arg(direction),
    sqlc.narg(source),
    sqlc.narg(destination),
    sqlc.arg(started_at),
    sqlc.narg(answered_at),
    sqlc.arg(ended_at),
    sqlc.arg(duration_seconds),
    sqlc.arg(billable_seconds),
    sqlc.arg(raw_payload),
    sqlc.arg(received_at)
FROM calls AS c
JOIN carrier_connections AS cc
  ON cc.id = sqlc.arg(carrier_connection_id)
JOIN carrier_providers AS cp
  ON cp.id = sqlc.arg(provider_id)
 AND cp.id = cc.provider_id
WHERE c.id = sqlc.arg(call_id)
  AND c.carrier_connection_id = cc.id
  AND c.state IN ('completed', 'failed', 'cancelled')
  AND c.ended_at IS NOT NULL
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.status = 'active'
  AND cp.status = 'active'
ON CONFLICT (provider_id, provider_cdr_id) DO NOTHING
RETURNING *;

-- name: GetProviderCDRByID :one
SELECT *
FROM provider_cdrs
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: GetProviderCDRByProviderRecord :one
SELECT *
FROM provider_cdrs
WHERE provider_id = sqlc.arg(provider_id)
  AND provider_cdr_id = sqlc.arg(provider_cdr_id)
LIMIT 1;

-- name: ListProviderCDRsByCall :many
SELECT *
FROM provider_cdrs
WHERE call_id = sqlc.arg(call_id)
ORDER BY started_at ASC, id ASC;

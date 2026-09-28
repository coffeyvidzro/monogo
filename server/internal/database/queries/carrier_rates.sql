-- name: CreateCarrierRate :one
INSERT INTO carrier_rates (
    organization_id,
    destination_prefix,
    direction,
    currency,
    rate_micros,
    billing_unit,
    billing_increment_seconds,
    minimum_duration_seconds,
    effective_at,
    expires_at
)
SELECT
    sqlc.narg(organization_id)::UUID,
    sqlc.arg(destination_prefix),
    sqlc.arg(direction),
    sqlc.arg(currency),
    sqlc.arg(rate_micros),
    sqlc.arg(billing_unit),
    sqlc.arg(billing_increment_seconds),
    sqlc.arg(minimum_duration_seconds),
    sqlc.arg(effective_at),
    sqlc.narg(expires_at)
WHERE sqlc.narg(organization_id)::UUID IS NULL
   OR EXISTS (
       SELECT 1
       FROM organizations AS o
       WHERE o.id = sqlc.narg(organization_id)::UUID
         AND o.status = 'active'
         AND o.deleted_at IS NULL
   )
RETURNING *;

-- name: GetCarrierRateByID :one
SELECT *
FROM carrier_rates
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: ResolveCarrierRate :one
SELECT cr.*
FROM carrier_rates AS cr
JOIN organizations AS o
  ON o.id = sqlc.arg(organization_id)
 AND o.status = 'active'
 AND o.deleted_at IS NULL
WHERE (cr.organization_id = o.id OR cr.organization_id IS NULL)
  AND cr.direction = sqlc.arg(direction)
  AND cr.currency = sqlc.arg(currency)
  AND sqlc.arg(destination_digits)::TEXT LIKE cr.destination_prefix || '%'
  AND cr.effective_at <= sqlc.arg(resolved_at)
  AND (cr.expires_at IS NULL OR cr.expires_at > sqlc.arg(resolved_at))
ORDER BY
    (cr.organization_id IS NOT NULL) DESC,
    length(cr.destination_prefix) DESC,
    cr.effective_at DESC
LIMIT 1;

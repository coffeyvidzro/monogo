-- name: CreateVoiceRate :one
INSERT INTO voice_rates (
    organization_id,
    destination_prefix,
    direction,
    currency,
    rate_micros,
    effective_at,
    expires_at
)
SELECT
    sqlc.narg(organization_id)::UUID,
    sqlc.arg(destination_prefix),
    sqlc.arg(direction),
    sqlc.arg(currency),
    sqlc.arg(rate_micros),
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

-- name: GetVoiceRateByID :one
SELECT *
FROM voice_rates
WHERE id = sqlc.arg(id)
LIMIT 1;

-- name: ResolveVoiceRate :one
SELECT vr.*
FROM voice_rates AS vr
JOIN organizations AS o
  ON o.id = sqlc.arg(organization_id)
 AND o.status = 'active'
 AND o.deleted_at IS NULL
WHERE (vr.organization_id = o.id OR vr.organization_id IS NULL)
  AND vr.direction = sqlc.arg(direction)
  AND vr.currency = sqlc.arg(currency)
  AND sqlc.arg(destination_digits)::TEXT LIKE vr.destination_prefix || '%'
  AND vr.effective_at <= sqlc.arg(resolved_at)
  AND (vr.expires_at IS NULL OR vr.expires_at > sqlc.arg(resolved_at))
ORDER BY
    (vr.organization_id IS NOT NULL) DESC,
    length(vr.destination_prefix) DESC,
    vr.effective_at DESC
LIMIT 1;

-- name: ListEntitlements :many
SELECT *
FROM entitlements
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY capability;

-- name: GetEntitlement :one
SELECT *
FROM entitlements
WHERE organization_id = sqlc.arg(organization_id)
  AND capability = sqlc.arg(capability)
LIMIT 1;

-- name: UpsertEntitlement :one
INSERT INTO entitlements (
    organization_id,
    capability,
    enabled
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(capability),
    sqlc.arg(enabled)
)
ON CONFLICT (organization_id, capability)
DO UPDATE SET enabled = EXCLUDED.enabled
RETURNING *;

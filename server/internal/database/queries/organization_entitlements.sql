-- name: ListOrganizationEntitlements :many
SELECT *
FROM organization_entitlements
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY capability;

-- name: GetOrganizationEntitlement :one
SELECT *
FROM organization_entitlements
WHERE organization_id = sqlc.arg(organization_id)
  AND capability = sqlc.arg(capability)
LIMIT 1;

-- name: UpsertOrganizationEntitlement :one
INSERT INTO organization_entitlements (
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

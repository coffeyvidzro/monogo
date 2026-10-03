-- name: ListRetentionPolicies :many
SELECT *
FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY resource;

-- name: GetRetentionPolicy :one
SELECT *
FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
  AND resource = sqlc.arg(resource)
LIMIT 1;

-- name: UpsertRetentionPolicy :one
INSERT INTO retention_policies (
    organization_id,
    resource,
    retention_days,
    enabled
)
VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(resource),
    sqlc.arg(retention_days),
    sqlc.arg(enabled)
)
ON CONFLICT (organization_id, resource)
DO UPDATE SET
    retention_days = EXCLUDED.retention_days,
    enabled = EXCLUDED.enabled
RETURNING *;

-- name: DeleteRetentionPolicy :exec
DELETE FROM retention_policies
WHERE organization_id = sqlc.arg(organization_id)
  AND resource = sqlc.arg(resource);

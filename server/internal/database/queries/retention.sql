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

-- name: ListEnabledRetentionPolicies :many
SELECT *
FROM retention_policies
WHERE enabled
ORDER BY organization_id, resource;

-- name: ListExpiredRecordingsForRetention :many
SELECT id
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND status = 'completed'
  AND created_at < sqlc.arg(created_before)
ORDER BY created_at
LIMIT sqlc.arg(batch_size);

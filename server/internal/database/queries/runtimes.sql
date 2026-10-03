-- name: CreateRuntime :one
INSERT INTO runtimes (
    organization_id,
    name,
    region
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.narg(region)
)
RETURNING *;

-- name: GetRuntime :one
SELECT *
FROM runtimes
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id);

-- name: ListRuntimes :many
SELECT *
FROM runtimes
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at ASC;

-- name: HeartbeatRuntime :one
UPDATE runtimes
SET
    version = sqlc.arg(version),
    capabilities = sqlc.arg(capabilities),
    capacity = sqlc.arg(capacity),
    active_sessions = sqlc.arg(active_sessions),
    last_seen_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'active'
RETURNING *;

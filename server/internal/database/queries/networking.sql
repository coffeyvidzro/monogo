-- name: CreateNetworkPolicy :one
INSERT INTO network_policies (
    id,
    organization_id,
    name,
    action,
    source_cidr
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.arg(action),
    sqlc.arg(source_cidr)
)
RETURNING *;

-- name: ListNetworkPolicies :many
SELECT *
FROM network_policies
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: GetNetworkPolicy :one
SELECT *
FROM network_policies
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: UpdateNetworkPolicy :one
UPDATE network_policies
SET name = COALESCE(sqlc.narg(name), name),
    action = COALESCE(sqlc.narg(action), action),
    source_cidr = COALESCE(sqlc.narg(source_cidr), source_cidr),
    status = COALESCE(sqlc.narg(status), status)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: DeleteNetworkPolicy :exec
DELETE FROM network_policies
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);

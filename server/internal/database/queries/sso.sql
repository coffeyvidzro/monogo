-- name: CreateSSOConnection :one
INSERT INTO sso_connections (
    id,
    organization_id,
    name,
    protocol,
    issuer,
    configuration,
    secret_ciphertext
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.arg(protocol),
    sqlc.arg(issuer),
    sqlc.arg(configuration),
    sqlc.narg(secret_ciphertext)
)
RETURNING *;

-- name: ListSSOConnections :many
SELECT id, organization_id, name, protocol, issuer, configuration, status,
       created_at, updated_at
FROM sso_connections
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: GetSSOConnection :one
SELECT *
FROM sso_connections
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: UpdateSSOConnection :one
UPDATE sso_connections
SET name = COALESCE(sqlc.narg(name), name),
    issuer = COALESCE(sqlc.narg(issuer), issuer),
    configuration = COALESCE(sqlc.narg(configuration), configuration),
    secret_ciphertext = COALESCE(sqlc.narg(secret_ciphertext), secret_ciphertext),
    status = COALESCE(sqlc.narg(status), status)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: DeleteSSOConnection :exec
DELETE FROM sso_connections
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);

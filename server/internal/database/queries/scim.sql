-- name: CreateSCIMToken :one
INSERT INTO scim_tokens (
    id,
    organization_id,
    name,
    token_prefix,
    token_hash,
    expires_at
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.arg(token_prefix),
    sqlc.arg(token_hash),
    sqlc.narg(expires_at)
)
RETURNING id, organization_id, name, token_prefix, expires_at, last_used_at,
          revoked_at, created_at, updated_at;

-- name: ListSCIMTokens :many
SELECT id, organization_id, name, token_prefix, expires_at, last_used_at,
       revoked_at, created_at, updated_at
FROM scim_tokens
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: GetActiveSCIMTokenByHash :one
SELECT *
FROM scim_tokens
WHERE token_hash = sqlc.arg(token_hash)
  AND revoked_at IS NULL
  AND (expires_at IS NULL OR expires_at > now())
LIMIT 1;

-- name: TouchSCIMToken :exec
UPDATE scim_tokens
SET last_used_at = now()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND revoked_at IS NULL;

-- name: RevokeSCIMToken :one
UPDATE scim_tokens
SET revoked_at = now()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND revoked_at IS NULL
RETURNING id, organization_id, name, token_prefix, expires_at, last_used_at,
          revoked_at, created_at, updated_at;

-- name: UpsertSCIMIdentity :one
INSERT INTO scim_identities (
    id,
    organization_id,
    user_id,
    external_id
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(user_id),
    sqlc.arg(external_id)
)
ON CONFLICT (organization_id, external_id)
DO UPDATE SET user_id = EXCLUDED.user_id
RETURNING *;

-- name: GetSCIMIdentityByExternalID :one
SELECT *
FROM scim_identities
WHERE organization_id = sqlc.arg(organization_id)
  AND external_id = sqlc.arg(external_id)
LIMIT 1;

-- name: DeleteSCIMIdentity :exec
DELETE FROM scim_identities
WHERE organization_id = sqlc.arg(organization_id)
  AND external_id = sqlc.arg(external_id);

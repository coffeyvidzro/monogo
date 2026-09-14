-- name: CreateOrganizationToken :one
INSERT INTO organization_tokens (
    organization_id,
    created_by,
    name,
    description,
    token_hash,
    token_prefix,
    scopes,
    expires_at
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(created_by),
    sqlc.arg(name),
    sqlc.narg(description),
    sqlc.arg(token_hash),
    sqlc.arg(token_prefix),
    sqlc.arg(scopes)::jsonb,
    sqlc.narg(expires_at)::timestamptz
FROM organizations AS o
JOIN users AS u ON u.id = sqlc.arg(created_by)
WHERE o.id = sqlc.arg(organization_id)
AND o.status = 'active'
AND o.deleted_at IS NULL
AND u.disabled_at IS NULL
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = o.id
    AND actor.user_id = sqlc.arg(created_by)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
RETURNING *;

-- name: GetOrganizationTokenByTokenHash :one
SELECT ot.*
FROM organization_tokens AS ot
JOIN organizations AS o ON o.id = ot.organization_id
WHERE ot.token_hash = sqlc.arg(token_hash)
AND ot.disabled_at IS NULL
AND (ot.expires_at IS NULL OR ot.expires_at > NOW())
AND o.status = 'active'
AND o.deleted_at IS NULL
LIMIT 1;

-- name: GetOrganizationTokenByID :one
SELECT ot.*
FROM organization_tokens AS ot
JOIN organizations AS o ON o.id = ot.organization_id
WHERE ot.id = sqlc.arg(id)
AND ot.organization_id = sqlc.arg(organization_id)
AND o.status = 'active'
AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListOrganizationTokensByOrganizationID :many
SELECT
    ot.id,
    ot.name,
    ot.description,
    ot.token_prefix,
    ot.scopes,
    ot.last_used_at,
    ot.last_used_ip,
    ot.expires_at,
    ot.created_at,
    u.name AS created_by
FROM organization_tokens AS ot
JOIN organizations AS o ON o.id = ot.organization_id
LEFT JOIN users AS u ON u.id = ot.created_by
WHERE ot.organization_id = sqlc.arg(organization_id)
AND o.status = 'active'
AND o.deleted_at IS NULL
ORDER BY ot.created_at DESC;

-- name: UpdateOrganizationToken :one
UPDATE organization_tokens AS target
SET
    name = COALESCE(sqlc.narg(name), target.name),
    description = COALESCE(sqlc.narg(description), target.description),
    scopes = COALESCE(sqlc.narg(scopes)::jsonb, target.scopes),
    expires_at = COALESCE(sqlc.narg(expires_at), target.expires_at),
    updated_at = NOW()
FROM organizations AS o
WHERE target.id = sqlc.arg(id)
AND target.organization_id = sqlc.arg(organization_id)
AND o.id = target.organization_id
AND o.status = 'active'
AND o.deleted_at IS NULL
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
RETURNING target.*;

-- name: DisableOrganizationToken :exec
UPDATE organization_tokens AS target
SET
    disabled_at = NOW(),
    updated_at = NOW()
WHERE target.id = sqlc.arg(id)
AND target.organization_id = sqlc.arg(organization_id)
AND target.disabled_at IS NULL
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
);

-- name: TouchOrganizationToken :exec
UPDATE organization_tokens
SET
    last_used_at = NOW(),
    last_used_ip = sqlc.narg(last_used_ip)::inet,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND disabled_at IS NULL
AND (expires_at IS NULL OR expires_at > NOW())
AND (last_used_at IS NULL OR last_used_at < NOW() - INTERVAL '1 minute');

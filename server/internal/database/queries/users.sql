-- name: CreateUser :one
INSERT INTO users (
    name,
    email,
    password_hash
) VALUES (
    sqlc.arg(name),
    sqlc.arg(email),
    sqlc.narg(password_hash)
)
RETURNING *;


-- name: GetUserByID :one
SELECT *
FROM users
WHERE id = sqlc.arg(id)
  AND disabled_at IS NULL
LIMIT 1;


-- name: GetUserByEmail :one
SELECT *
FROM users
WHERE email = sqlc.arg(email)
  AND disabled_at IS NULL
LIMIT 1;


-- name: GetUserByEmailIncludingDisabled :one
SELECT *
FROM users
WHERE email = sqlc.arg(email)
LIMIT 1;


-- name: SetUserPassword :one
UPDATE users
SET
    password_hash = sqlc.arg(password_hash),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND disabled_at IS NULL
RETURNING *;


-- name: MarkUserEmailVerified :one
UPDATE users
SET
    email_verified = TRUE,
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND disabled_at IS NULL
RETURNING *;


-- name: DisableUser :exec
UPDATE users
SET
    disabled_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND disabled_at IS NULL;

-- name: UpdateUserProfile :one
UPDATE users
SET
    name = COALESCE(sqlc.narg(name), name),
    email = COALESCE(sqlc.narg(email), email),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND disabled_at IS NULL
RETURNING *;

-- name: ListBackofficeUsers :many
SELECT
    u.id::TEXT AS id,
    COALESCE(NULLIF(BTRIM(u.name), ''), '—')::TEXT AS name,
    u.email::TEXT AS email,
    u.email_verified,
    u.is_platform_admin,
    CASE WHEN u.disabled_at IS NULL THEN 'active' ELSE 'disabled' END::TEXT AS status,
    COUNT(om.organization_id) FILTER (WHERE om.status = 'active')::BIGINT AS organization_count,
    to_char(u.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS created_at
FROM users AS u
LEFT JOIN organization_members AS om ON om.user_id = u.id
GROUP BY
    u.id,
    u.name,
    u.email,
    u.email_verified,
    u.is_platform_admin,
    u.disabled_at,
    u.created_at
ORDER BY u.created_at DESC
LIMIT 100;

-- name: GetBackofficeUser :one
SELECT
    u.id::TEXT AS id,
    COALESCE(NULLIF(BTRIM(u.name), ''), '—')::TEXT AS name,
    u.email::TEXT AS email,
    u.email_verified,
    u.is_platform_admin,
    CASE WHEN u.disabled_at IS NULL THEN 'active' ELSE 'disabled' END::TEXT AS status,
    COUNT(DISTINCT om.organization_id) FILTER (WHERE om.status = 'active')::BIGINT AS organization_count,
    COUNT(DISTINCT s.id) FILTER (
        WHERE s.revoked_at IS NULL
          AND s.expires_at > NOW()
    )::BIGINT AS active_session_count,
    COALESCE(
        to_char(MAX(s.last_seen_at) AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'),
        '—'
    )::TEXT AS last_seen_at,
    to_char(u.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS created_at,
    to_char(u.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS updated_at,
    COALESCE(
        to_char(u.disabled_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'),
        '—'
    )::TEXT AS disabled_at
FROM users AS u
LEFT JOIN organization_members AS om ON om.user_id = u.id
LEFT JOIN sessions AS s ON s.user_id = u.id
WHERE u.id = sqlc.arg(id)
GROUP BY
    u.id,
    u.name,
    u.email,
    u.email_verified,
    u.is_platform_admin,
    u.disabled_at,
    u.created_at,
    u.updated_at
LIMIT 1;

-- name: ListBackofficeUserOrganizations :many
SELECT
    o.id::TEXT AS organization_id,
    o.name,
    CASE WHEN o.deleted_at IS NULL THEN o.status ELSE 'deleted' END::TEXT AS organization_status,
    om.role,
    om.status AS membership_status,
    to_char(om.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS joined_at
FROM organization_members AS om
JOIN organizations AS o ON o.id = om.organization_id
WHERE om.user_id = sqlc.arg(user_id)
ORDER BY
    CASE om.role WHEN 'owner' THEN 0 WHEN 'admin' THEN 1 ELSE 2 END,
    om.created_at ASC;

-- name: ListBackofficeUserSessions :many
SELECT
    s.id::TEXT AS session_id,
    s.assurance,
    COALESCE(host(s.ip_address), '—')::TEXT AS ip_address,
    COALESCE(NULLIF(BTRIM(s.user_agent), ''), '—')::TEXT AS user_agent,
    CASE
        WHEN s.revoked_at IS NOT NULL THEN 'revoked'
        WHEN s.expires_at <= NOW() THEN 'expired'
        ELSE 'active'
    END::TEXT AS status,
    to_char(s.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS created_at,
    COALESCE(
        to_char(s.last_seen_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'),
        '—'
    )::TEXT AS last_seen_at,
    to_char(s.expires_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI')::TEXT AS expires_at,
    COALESCE(
        to_char(s.revoked_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI'),
        '—'
    )::TEXT AS revoked_at
FROM sessions AS s
WHERE s.user_id = sqlc.arg(user_id)
ORDER BY s.created_at DESC
LIMIT 20;

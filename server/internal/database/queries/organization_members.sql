-- name: AddOrganizationMember :one
INSERT INTO organization_members (
    organization_id,
    user_id,
    role
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(user_id),
    sqlc.arg(role)
FROM users AS u
WHERE u.id = sqlc.arg(user_id)
AND u.disabled_at IS NULL
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = sqlc.arg(organization_id)
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
RETURNING *;

-- name: GetOrganizationMember :one
SELECT tm.*
FROM organization_members AS tm
JOIN users AS u ON u.id = tm.user_id
JOIN organizations AS o ON o.id = tm.organization_id
WHERE tm.organization_id = sqlc.arg(organization_id)
AND tm.user_id = sqlc.arg(user_id)
AND tm.status = 'active'
AND u.disabled_at IS NULL
AND o.status = 'active'
AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListMembersByOrganizationID :many
SELECT tm.*
FROM organization_members AS tm
JOIN users AS u ON u.id = tm.user_id
WHERE tm.organization_id = sqlc.arg(organization_id)
AND tm.status = 'active'
AND u.disabled_at IS NULL
ORDER BY tm.created_at ASC;

-- name: ListOrganizationMembershipsByUserID :many
SELECT tm.*
FROM organization_members AS tm
JOIN organizations AS t ON t.id = tm.organization_id
WHERE tm.user_id = sqlc.arg(user_id)
AND tm.status = 'active'
AND t.status = 'active'
AND t.deleted_at IS NULL
ORDER BY tm.created_at ASC;

-- name: UpdateMemberRole :one
UPDATE organization_members AS target
SET
    role = sqlc.arg(role),
    updated_at = NOW()
WHERE target.organization_id = sqlc.arg(organization_id)
AND target.user_id = sqlc.arg(user_id)
AND target.status = 'active'
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
AND target.role <> 'owner'
RETURNING target.*;

-- name: DisableOrganizationMember :exec
UPDATE organization_members AS target
SET
    status = 'disabled',
    updated_at = NOW()
WHERE target.organization_id = sqlc.arg(organization_id)
AND target.user_id = sqlc.arg(user_id)
AND target.status = 'active'
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
AND target.role <> 'owner';

-- name: EnableOrganizationMember :exec
UPDATE organization_members AS target
SET
    status = 'active',
    updated_at = NOW()
WHERE target.organization_id = sqlc.arg(organization_id)
AND target.user_id = sqlc.arg(user_id)
AND target.status = 'disabled'
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
AND target.role <> 'owner';

-- name: RemoveOrganizationMember :exec
DELETE FROM organization_members AS target
WHERE target.organization_id = sqlc.arg(organization_id)
AND target.user_id = sqlc.arg(user_id)
AND EXISTS (
    SELECT 1
    FROM organization_members AS actor
    WHERE actor.organization_id = target.organization_id
    AND actor.user_id = sqlc.arg(actor_user_id)
    AND actor.status = 'active'
    AND actor.role IN ('owner', 'admin')
)
AND target.role <> 'owner';

-- name: CreateOrganization :one
INSERT INTO organizations (
    name
) VALUES (
    sqlc.arg(name)
)
RETURNING *;

-- name: CreateOrganizationWithOwner :one
WITH new_organization AS (
    INSERT INTO organizations (
        name
    )
    SELECT sqlc.arg(name)
    FROM users AS u
    WHERE u.id = sqlc.arg(user_id)
    AND u.disabled_at IS NULL
    RETURNING *
), owner_membership AS (
    INSERT INTO organization_members (
        organization_id,
        user_id,
        role
    )
    SELECT
        o.id,
        sqlc.arg(user_id),
        'owner'
    FROM new_organization AS o
    RETURNING organization_id
)
SELECT o.*
FROM new_organization AS o
JOIN owner_membership AS om ON om.organization_id = o.id;

-- name: GetOrganizationByID :one
SELECT *
FROM organizations
WHERE id = sqlc.arg(id)
AND status = 'active'
AND deleted_at IS NULL
LIMIT 1;

-- name: UpdateOrganization :one
UPDATE organizations
SET
    name = COALESCE(sqlc.narg(name), name),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND status = 'active'
AND deleted_at IS NULL
RETURNING *;

-- name: DeleteOrganization :exec
UPDATE organizations
SET
    status = 'disabled',
    deleted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND deleted_at IS NULL;

-- name: ListOrganizationsByUserID :many
SELECT t.*, tm.role AS member_role
FROM organizations AS t
JOIN organization_members AS tm ON tm.organization_id = t.id
JOIN users AS u ON u.id = tm.user_id
WHERE tm.user_id = sqlc.arg(user_id)
AND u.disabled_at IS NULL
AND tm.status = 'active'
AND t.status = 'active'
AND t.deleted_at IS NULL
ORDER BY t.created_at DESC;




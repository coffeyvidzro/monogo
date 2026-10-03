-- name: CreateSCIMUser :one
WITH existing_user AS (
    SELECT users.id
    FROM users
    WHERE users.email = sqlc.arg(user_name)
), created_user AS (
    INSERT INTO users (name, email, email_verified)
    SELECT sqlc.narg(display_name), sqlc.arg(user_name), TRUE
    WHERE NOT EXISTS (SELECT 1 FROM existing_user)
    RETURNING id
), target_user AS (
    SELECT id FROM existing_user
    UNION ALL
    SELECT id FROM created_user
), membership AS (
    INSERT INTO organization_members (
        organization_id,
        user_id,
        role,
        status
    )
    SELECT
        sqlc.arg(organization_id),
        target_user.id,
        'member',
        CASE WHEN sqlc.arg(active)::boolean THEN 'active' ELSE 'disabled' END
    FROM target_user
    ON CONFLICT (organization_id, user_id)
    DO UPDATE SET
        status = CASE
            WHEN organization_members.role = 'owner' THEN organization_members.status
            ELSE EXCLUDED.status
        END,
        updated_at = NOW()
    RETURNING user_id
), identity AS (
    INSERT INTO scim_identities (
        id,
        organization_id,
        user_id,
        external_id
    )
    SELECT
        sqlc.arg(id),
        sqlc.arg(organization_id),
        membership.user_id,
        sqlc.arg(external_id)
    FROM membership
    RETURNING id, organization_id, user_id, external_id, created_at, updated_at
), profile AS (
    INSERT INTO scim_user_profiles (
        identity_id,
        organization_id,
        user_id,
        user_name,
        display_name,
        active
    )
    SELECT
        identity.id,
        identity.organization_id,
        identity.user_id,
        sqlc.arg(user_name),
        sqlc.narg(display_name),
        sqlc.arg(active)
    FROM identity
    RETURNING *
)
SELECT
    profile.identity_id AS id,
    profile.organization_id,
    profile.user_id,
    identity.external_id,
    profile.user_name::TEXT AS user_name,
    profile.display_name,
    profile.active,
    profile.created_at,
    profile.updated_at
FROM profile
JOIN identity ON identity.id = profile.identity_id;

-- name: GetSCIMUser :one
SELECT
    profile.identity_id AS id,
    profile.organization_id,
    profile.user_id,
    identity.external_id,
    profile.user_name::TEXT AS user_name,
    profile.display_name,
    profile.active,
    profile.created_at,
    profile.updated_at
FROM scim_user_profiles AS profile
JOIN scim_identities AS identity
  ON identity.id = profile.identity_id
 AND identity.organization_id = profile.organization_id
WHERE profile.organization_id = sqlc.arg(organization_id)
  AND profile.identity_id = sqlc.arg(id)
LIMIT 1;

-- name: GetSCIMUserByUserName :one
SELECT
    profile.identity_id AS id,
    profile.organization_id,
    profile.user_id,
    identity.external_id,
    profile.user_name::TEXT AS user_name,
    profile.display_name,
    profile.active,
    profile.created_at,
    profile.updated_at
FROM scim_user_profiles AS profile
JOIN scim_identities AS identity
  ON identity.id = profile.identity_id
 AND identity.organization_id = profile.organization_id
WHERE profile.organization_id = sqlc.arg(organization_id)
  AND profile.user_name = sqlc.arg(user_name)
LIMIT 1;

-- name: ListSCIMUsers :many
SELECT
    profile.identity_id AS id,
    profile.organization_id,
    profile.user_id,
    identity.external_id,
    profile.user_name::TEXT AS user_name,
    profile.display_name,
    profile.active,
    profile.created_at,
    profile.updated_at
FROM scim_user_profiles AS profile
JOIN scim_identities AS identity
  ON identity.id = profile.identity_id
 AND identity.organization_id = profile.organization_id
WHERE profile.organization_id = sqlc.arg(organization_id)
ORDER BY profile.created_at, profile.identity_id
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountSCIMUsers :one
SELECT COUNT(*)::BIGINT
FROM scim_user_profiles AS profile
WHERE profile.organization_id = sqlc.arg(organization_id);

-- name: ReplaceSCIMUser :one
WITH target AS (
    SELECT profile.identity_id, profile.user_id
    FROM scim_user_profiles AS profile
    WHERE profile.organization_id = sqlc.arg(organization_id)
      AND profile.identity_id = sqlc.arg(id)
), membership AS (
    UPDATE organization_members AS member
    SET
        status = CASE
            WHEN member.role = 'owner' THEN member.status
            WHEN sqlc.arg(active)::boolean THEN 'active'
            ELSE 'disabled'
        END,
        updated_at = NOW()
    FROM target
    WHERE member.organization_id = sqlc.arg(organization_id)
      AND member.user_id = target.user_id
    RETURNING member.user_id
), updated AS (
    UPDATE scim_user_profiles AS profile
    SET
        user_name = sqlc.arg(user_name),
        display_name = sqlc.narg(display_name),
        active = sqlc.arg(active)
    FROM target
    WHERE profile.organization_id = sqlc.arg(organization_id)
      AND profile.identity_id = target.identity_id
    RETURNING profile.*
)
SELECT
    updated.identity_id AS id,
    updated.organization_id,
    updated.user_id,
    identity.external_id,
    updated.user_name::TEXT AS user_name,
    updated.display_name,
    updated.active,
    updated.created_at,
    updated.updated_at
FROM updated
JOIN scim_identities AS identity
  ON identity.id = updated.identity_id
 AND identity.organization_id = updated.organization_id;

-- name: DeleteSCIMUser :one
WITH target AS (
    SELECT profile.identity_id, profile.user_id
    FROM scim_user_profiles AS profile
    JOIN organization_members AS member
      ON member.organization_id = profile.organization_id
     AND member.user_id = profile.user_id
    WHERE profile.organization_id = sqlc.arg(organization_id)
      AND profile.identity_id = sqlc.arg(id)
      AND member.role <> 'owner'
), disabled AS (
    UPDATE organization_members AS member
    SET status = 'disabled', updated_at = NOW()
    FROM target
    WHERE member.organization_id = sqlc.arg(organization_id)
      AND member.user_id = target.user_id
    RETURNING member.user_id
), deleted AS (
    DELETE FROM scim_identities AS identity
    USING target
    WHERE identity.organization_id = sqlc.arg(organization_id)
      AND identity.id = target.identity_id
    RETURNING identity.id
)
SELECT id FROM deleted;

-- name: CreateSCIMGroup :one
INSERT INTO scim_groups (
    id,
    organization_id,
    external_id,
    display_name
)
VALUES (
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.narg(external_id),
    sqlc.arg(display_name)
)
RETURNING *;

-- name: GetSCIMGroup :one
SELECT *
FROM scim_groups
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetSCIMGroupByDisplayName :one
SELECT *
FROM scim_groups
WHERE organization_id = sqlc.arg(organization_id)
  AND display_name = sqlc.arg(display_name)
LIMIT 1;

-- name: ListSCIMGroups :many
SELECT *
FROM scim_groups
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at, id
LIMIT sqlc.arg(limit_count)
OFFSET sqlc.arg(offset_count);

-- name: CountSCIMGroups :one
SELECT COUNT(*)::BIGINT
FROM scim_groups
WHERE organization_id = sqlc.arg(organization_id);

-- name: ReplaceSCIMGroup :one
UPDATE scim_groups
SET
    external_id = sqlc.narg(external_id),
    display_name = sqlc.arg(display_name)
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: DeleteSCIMGroup :exec
DELETE FROM scim_groups
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id);

-- name: ListSCIMGroupMembers :many
SELECT
    profile.identity_id AS id,
    profile.user_name::TEXT AS user_name,
    profile.display_name
FROM scim_group_members AS member
JOIN scim_user_profiles AS profile
  ON profile.organization_id = member.organization_id
 AND profile.identity_id = member.identity_id
WHERE member.organization_id = sqlc.arg(organization_id)
  AND member.group_id = sqlc.arg(group_id)
ORDER BY profile.user_name;

-- name: ReplaceSCIMGroupMembers :many
WITH cleared AS (
    DELETE FROM scim_group_members AS member
    WHERE member.organization_id = sqlc.arg(organization_id)
      AND member.group_id = sqlc.arg(group_id)
), requested AS (
    SELECT unnest(sqlc.arg(member_ids)::UUID[]) AS identity_id
), inserted AS (
    INSERT INTO scim_group_members (
        organization_id,
        group_id,
        identity_id
    )
    SELECT
        sqlc.arg(organization_id),
        sqlc.arg(group_id),
        profile.identity_id
    FROM requested
    JOIN scim_user_profiles AS profile
      ON profile.organization_id = sqlc.arg(organization_id)
     AND profile.identity_id = requested.identity_id
    RETURNING identity_id
)
SELECT inserted.identity_id AS id
FROM inserted
ORDER BY inserted.identity_id;

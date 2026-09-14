-- name: CreateInvitation :one
INSERT INTO organization_invitations (
    organization_id,
    invited_by,
    email,
    role,
    token_hash,
    expires_at
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(invited_by),
    sqlc.arg(email),
    sqlc.arg(role),
    sqlc.arg(token_hash),
    sqlc.arg(expires_at)
FROM organizations AS t
JOIN users AS u ON u.id = sqlc.arg(invited_by)
JOIN organization_members AS tm ON tm.organization_id = t.id AND tm.user_id = u.id
WHERE t.id = sqlc.arg(organization_id)
AND t.status = 'active'
AND t.deleted_at IS NULL
AND u.disabled_at IS NULL
AND tm.status = 'active'
AND tm.role IN ('owner', 'admin')  -- Only owners and admins can invite
RETURNING *;

-- name: GetInvitationByTokenHash :one
SELECT i.*
FROM organization_invitations AS i
JOIN organizations AS t ON t.id = i.organization_id
WHERE i.token_hash = sqlc.arg(token_hash)
AND i.status = 'pending'
AND i.expires_at > NOW()
AND t.status = 'active'
AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetPendingInvitationByEmailAndOrganization :one
SELECT *
FROM organization_invitations
WHERE organization_id = sqlc.arg(organization_id)
AND email = sqlc.arg(email)
AND status = 'pending'
AND expires_at > NOW()
LIMIT 1;

-- name: ListPendingInvitationsByOrganizationID :many
SELECT i.*
FROM organization_invitations AS i
WHERE i.organization_id = sqlc.arg(organization_id)
AND i.status = 'pending'
AND i.expires_at > NOW()
ORDER BY i.created_at DESC;

-- name: ListInvitationsForEmail :many
SELECT i.*, t.name AS organization_name
FROM organization_invitations AS i
JOIN organizations AS t ON t.id = i.organization_id
WHERE i.email = sqlc.arg(email)
AND i.status = 'pending'
AND i.expires_at > NOW()
AND t.status = 'active'
AND t.deleted_at IS NULL
ORDER BY i.created_at DESC;

-- name: AcceptInvitation :one
UPDATE organization_invitations
SET
    status = 'accepted',
    accepted_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND status = 'pending'
AND expires_at > NOW()
RETURNING *;

-- name: DeclineInvitation :one
UPDATE organization_invitations
SET
    status = 'declined',
    declined_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND status = 'pending'
RETURNING *;

-- name: RevokeInvitation :one
UPDATE organization_invitations
SET
    status = 'revoked',
    revoked_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND organization_id = sqlc.arg(organization_id)
AND status = 'pending'
RETURNING *;

-- name: ExpireStaleInvitations :exec
UPDATE organization_invitations
SET
    status = 'expired',
    updated_at = NOW()
WHERE status = 'pending'
AND expires_at <= NOW();

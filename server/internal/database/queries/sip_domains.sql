-- name: CreateSipDomain :one
INSERT INTO sip_domains (
    organization_id,
    domain
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(domain)
FROM organizations AS t
WHERE t.id = sqlc.arg(organization_id)
AND t.status = 'active'
AND t.deleted_at IS NULL
RETURNING *;

-- name: GetSipDomainByID :one
SELECT sd.*
FROM sip_domains AS sd
JOIN organizations AS t ON t.id = sd.organization_id
WHERE sd.id = sqlc.arg(id)
AND sd.organization_id = sqlc.arg(organization_id)
AND sd.status = 'active'
AND t.status = 'active'
AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetSipDomainByDomain :one
SELECT sd.*
FROM sip_domains AS sd
JOIN organizations AS t ON t.id = sd.organization_id
WHERE sd.domain = sqlc.arg(domain)
AND sd.organization_id = sqlc.arg(organization_id)
AND sd.status = 'active'
AND t.status = 'active'
AND t.deleted_at IS NULL
LIMIT 1;

-- name: ListSipDomainsByOrganizationID :many
SELECT sd.*
FROM sip_domains AS sd
JOIN organizations AS t ON t.id = sd.organization_id
WHERE sd.organization_id = sqlc.arg(organization_id)
AND sd.status = 'active'
AND t.status = 'active'
AND t.deleted_at IS NULL
ORDER BY sd.created_at ASC;

-- name: UpdateSipDomain :one
UPDATE sip_domains
SET
    domain = COALESCE(sqlc.narg(domain), domain),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND organization_id = sqlc.arg(organization_id)
AND status = 'active'
RETURNING *;

-- name: DisableSipDomain :exec
UPDATE sip_domains
SET
    status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND organization_id = sqlc.arg(organization_id)
AND status = 'active';

-- name: EnableSipDomain :exec
UPDATE sip_domains
SET
    status = 'active',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
AND organization_id = sqlc.arg(organization_id)
AND status = 'disabled';
-- name: ListBackofficeSIPDomains :many
SELECT d.id::TEXT AS id, d.organization_id::TEXT AS organization_id, o.name AS organization_name,
       d.domain::TEXT AS domain, d.status, COUNT(s.id)::BIGINT AS subscriber_count,
       to_char(d.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM sip_domains d JOIN organizations o ON o.id=d.organization_id
LEFT JOIN subscribers s ON s.sip_domain_id=d.id
GROUP BY d.id,o.name ORDER BY d.created_at DESC LIMIT 100;

-- name: GetBackofficeSIPDomain :one
SELECT d.id::TEXT AS id, d.organization_id::TEXT AS organization_id, o.name AS organization_name,
       d.domain::TEXT AS domain, d.status, COUNT(DISTINCT s.id)::BIGINT AS subscriber_count,
       COUNT(DISTINCT vb.id)::BIGINT AS binding_count,
       to_char(d.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
       to_char(d.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM sip_domains d JOIN organizations o ON o.id=d.organization_id
LEFT JOIN subscribers s ON s.sip_domain_id=d.id LEFT JOIN voice_bindings vb ON vb.sip_domain_id=d.id
WHERE d.id=sqlc.arg(id) GROUP BY d.id,o.name LIMIT 1;

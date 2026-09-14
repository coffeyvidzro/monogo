-- name: CreateVoiceApplication :one
INSERT INTO voice_applications (
    organization_id,
    name,
    ring_timeout_seconds,
    caller_id,
    voice_url,
    callback_url
)
SELECT
    sqlc.arg(organization_id) as organization_id,
    sqlc.arg(name) as name,
    COALESCE(sqlc.narg(ring_timeout_seconds), 30) as ring_timeout_seconds,
    sqlc.narg(caller_id) as caller_id,
    sqlc.narg(voice_url) as voice_url,
    sqlc.narg(callback_url) as callback_url
FROM organizations AS t
WHERE t.id = sqlc.arg(organization_id)
  AND t.status = 'active'
  AND t.deleted_at IS NULL
RETURNING *;

-- name: GetVoiceApplicationByID :one
SELECT va.*
FROM voice_applications AS va
JOIN organizations AS t ON t.id = va.organization_id
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetVoiceApplicationByName :one
SELECT va.*
FROM voice_applications AS va
JOIN organizations AS t ON t.id = va.organization_id
WHERE va.organization_id = sqlc.arg(organization_id)
  AND va.name = sqlc.arg(name)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: ListVoiceApplicationsByOrganizationID :many
SELECT va.*
FROM voice_applications AS va
WHERE va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
ORDER BY va.created_at DESC;

-- name: UpdateVoiceApplication :one
UPDATE voice_applications
SET
    name = COALESCE(sqlc.narg(name), name),
    ring_timeout_seconds = COALESCE(sqlc.narg(ring_timeout_seconds), ring_timeout_seconds),
    caller_id = COALESCE(sqlc.narg(caller_id), caller_id),
    voice_url = COALESCE(sqlc.narg(voice_url), voice_url),
    callback_url = COALESCE(sqlc.narg(callback_url), callback_url),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: DisableVoiceApplication :exec
UPDATE voice_applications
SET
    status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active';

-- name: EnableVoiceApplication :exec
UPDATE voice_applications
SET
    status = 'active',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'disabled';

-- name: CreateVoiceBinding :one
INSERT INTO voice_bindings (
    voice_application_id,
    phone_number_id,
    sip_domain_id,
    subscriber_id
)
SELECT
    sqlc.arg(voice_application_id),
    sqlc.narg(phone_number_id)::uuid,
    sqlc.narg(sip_domain_id)::uuid,
    sqlc.narg(subscriber_id)::uuid
FROM voice_applications AS va
JOIN organizations AS t ON t.id = va.organization_id
WHERE va.id = sqlc.arg(voice_application_id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
RETURNING *;

-- name: GetVoiceBindingByID :one
SELECT vb.*
FROM voice_bindings AS vb
JOIN voice_applications AS va ON va.id = vb.voice_application_id
JOIN organizations AS t ON t.id = va.organization_id
WHERE vb.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetVoiceBindingByPhoneNumberID :one
SELECT vb.*
FROM voice_bindings AS vb
JOIN voice_applications AS va ON va.id = vb.voice_application_id
JOIN organizations AS t ON t.id = va.organization_id
WHERE vb.phone_number_id = sqlc.arg(phone_number_id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetVoiceBindingBySipDomainID :one
SELECT vb.*
FROM voice_bindings AS vb
JOIN voice_applications AS va ON va.id = vb.voice_application_id
JOIN organizations AS t ON t.id = va.organization_id
WHERE vb.sip_domain_id = sqlc.arg(sip_domain_id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: GetVoiceBindingBySubscriberID :one
SELECT vb.*
FROM voice_bindings AS vb
JOIN voice_applications AS va ON va.id = vb.voice_application_id
JOIN organizations AS t ON t.id = va.organization_id
WHERE vb.subscriber_id = sqlc.arg(subscriber_id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND t.status = 'active'
  AND t.deleted_at IS NULL
LIMIT 1;

-- name: ListVoiceBindingsByApplicationID :many
SELECT vb.*
FROM voice_bindings AS vb
JOIN voice_applications AS va ON va.id = vb.voice_application_id
WHERE vb.voice_application_id = sqlc.arg(voice_application_id)
  AND va.organization_id = sqlc.arg(organization_id)
ORDER BY vb.created_at DESC;

-- name: DeleteVoiceBinding :exec
DELETE FROM voice_bindings
WHERE voice_bindings.id = sqlc.arg(id)
  AND voice_bindings.voice_application_id = sqlc.arg(voice_application_id)
  AND voice_application_id IN (
      SELECT va.id FROM voice_applications AS va
      WHERE va.organization_id = sqlc.arg(organization_id)
  );
-- name: ListBackofficeVoiceApplications :many
SELECT va.id::TEXT AS id, va.organization_id::TEXT AS organization_id, o.name AS organization_name,
       va.name, va.status, va.ring_timeout_seconds, COUNT(vb.id)::BIGINT AS binding_count,
       to_char(va.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM voice_applications va JOIN organizations o ON o.id=va.organization_id
LEFT JOIN voice_bindings vb ON vb.voice_application_id=va.id
GROUP BY va.id,o.name ORDER BY va.created_at DESC LIMIT 100;

-- name: GetBackofficeVoiceApplication :one
SELECT va.id::TEXT AS id, va.organization_id::TEXT AS organization_id, o.name AS organization_name,
       va.name, va.status, va.ring_timeout_seconds,
       CAST(COALESCE(va.caller_id, '—') AS TEXT) AS caller_id,
       CAST(COALESCE(va.voice_url, '—') AS TEXT) AS voice_url,
       CAST(COALESCE(va.callback_url, '—') AS TEXT) AS callback_url,
       COUNT(vb.id)::BIGINT AS binding_count,
       to_char(va.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
       to_char(va.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM voice_applications va JOIN organizations o ON o.id=va.organization_id
LEFT JOIN voice_bindings vb ON vb.voice_application_id=va.id
WHERE va.id=sqlc.arg(id) GROUP BY va.id,o.name LIMIT 1;

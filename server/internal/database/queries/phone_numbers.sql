-- name: CreateBYOCPhoneNumber :one
INSERT INTO phone_numbers (
    organization_id,
    number,
    country_code,
    provisioning_mode,
    carrier_connection_id,
    provider_id,
    provider_resource_id,
    voice_enabled,
    sms_enabled
)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(number) AS number,
    sqlc.arg(country_code) AS country_code,
    'byoc' AS provisioning_mode,
    sqlc.narg(carrier_connection_id) AS carrier_connection_id,
    NULL::UUID AS provider_id,
    NULL::TEXT AS provider_resource_id,
    COALESCE(sqlc.narg(voice_enabled), true) AS voice_enabled,
    COALESCE(sqlc.narg(sms_enabled), false) AS sms_enabled
FROM organizations AS o
LEFT JOIN carrier_connections AS cc
  ON cc.id = sqlc.narg(carrier_connection_id)::UUID
 AND cc.scope = 'organization'
 AND cc.organization_id = sqlc.arg(organization_id)
 AND cc.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
  AND (
      sqlc.narg(carrier_connection_id)::UUID IS NULL
      OR cc.id IS NOT NULL
  )
RETURNING *;

-- name: CreateManagedPhoneNumber :one
INSERT INTO phone_numbers (
    organization_id,
    number,
    country_code,
    provisioning_mode,
    carrier_connection_id,
    provider_id,
    provider_resource_id,
    voice_enabled,
    sms_enabled
)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(number) AS number,
    sqlc.arg(country_code) AS country_code,
    'managed' AS provisioning_mode,
    sqlc.narg(carrier_connection_id) AS carrier_connection_id,
    sqlc.arg(provider_id) AS provider_id,
    sqlc.arg(provider_resource_id) AS provider_resource_id,
    COALESCE(sqlc.narg(voice_enabled), true) AS voice_enabled,
    COALESCE(sqlc.narg(sms_enabled), false) AS sms_enabled
FROM organizations AS o
JOIN carrier_providers AS cp
  ON cp.id = sqlc.arg(provider_id)
 AND cp.status = 'active'
LEFT JOIN carrier_connections AS cc
  ON cc.id = sqlc.narg(carrier_connection_id)::UUID
 AND cc.scope = 'platform'
 AND cc.organization_id IS NULL
 AND cc.provider_id = cp.id
 AND cc.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
  AND length(btrim(sqlc.arg(provider_resource_id))) > 0
  AND (
      sqlc.narg(carrier_connection_id)::UUID IS NULL
      OR cc.id IS NOT NULL
  )
RETURNING *;

-- name: CreateProvisioningManagedPhoneNumber :one
INSERT INTO phone_numbers (
    organization_id,
    number,
    country_code,
    provisioning_mode,
    carrier_connection_id,
    provider_id,
    provider_resource_id,
    voice_enabled,
    sms_enabled,
    status
)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(number) AS number,
    sqlc.arg(country_code) AS country_code,
    'managed' AS provisioning_mode,
    cc.id AS carrier_connection_id,
    cp.id AS provider_id,
    NULL::TEXT AS provider_resource_id,
    true AS voice_enabled,
    false AS sms_enabled,
    'provisioning' AS status
FROM organizations AS o
JOIN carrier_providers AS cp
  ON cp.id = sqlc.arg(provider_id)
 AND cp.status = 'active'
JOIN carrier_connections AS cc
  ON cc.id = sqlc.arg(carrier_connection_id)
 AND cc.scope = 'platform'
 AND cc.organization_id IS NULL
 AND cc.provider_id = cp.id
 AND cc.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: LockManagedPhoneNumberForProviderOperation :one
SELECT *
FROM phone_numbers
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND provider_id = sqlc.arg(provider_id)
  AND provisioning_mode = 'managed'
  AND number = sqlc.arg(number)
  AND country_code = sqlc.arg(country_code)
FOR UPDATE;

-- name: MarkManagedPhoneNumberActive :one
UPDATE phone_numbers
SET
    provider_resource_id = sqlc.arg(provider_resource_id),
    status = 'active',
    error_code = NULL,
    error_message = NULL
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND provider_id = sqlc.arg(provider_id)
  AND provisioning_mode = 'managed'
  AND status = 'provisioning'
RETURNING *;

-- name: MarkManagedPhoneNumberFailed :exec
UPDATE phone_numbers
SET
    status = 'failed',
    voice_enabled = false,
    sms_enabled = false,
    error_code = 'provider_provision_failed',
    error_message = sqlc.arg(error_message)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND provider_id = sqlc.arg(provider_id)
  AND provisioning_mode = 'managed'
  AND status = 'provisioning';

-- name: GetPhoneNumberByID :one
SELECT pn.*
FROM phone_numbers AS pn
JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.id = sqlc.arg(id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status <> 'released'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: GetPhoneNumberForRelease :one
SELECT pn.*
FROM phone_numbers AS pn
JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.id = sqlc.arg(id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status IN ('active', 'disabled')
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: GetPhoneNumberByNumber :one
SELECT pn.*
FROM phone_numbers AS pn
JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.number = sqlc.arg(number)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListPhoneNumbersByOrganizationID :many
SELECT pn.*
FROM phone_numbers AS pn
WHERE pn.organization_id = sqlc.arg(organization_id)
  AND pn.status <> 'released'
ORDER BY pn.created_at DESC;

-- name: ListPhoneNumbersByCountry :many
SELECT pn.*
FROM phone_numbers AS pn
WHERE pn.organization_id = sqlc.arg(organization_id)
  AND pn.country_code = sqlc.arg(country_code)
  AND pn.status <> 'released'
ORDER BY pn.number ASC;

-- name: UpdatePhoneNumber :one
UPDATE phone_numbers AS pn
SET
    voice_enabled = COALESCE(sqlc.narg(voice_enabled), pn.voice_enabled),
    sms_enabled = COALESCE(sqlc.narg(sms_enabled), pn.sms_enabled),
    updated_at = NOW()
WHERE pn.id = sqlc.arg(id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status = 'active'
RETURNING pn.*;

-- name: SetBYOCPhoneNumberCarrierConnection :one
UPDATE phone_numbers AS pn
SET
    carrier_connection_id = sqlc.arg(carrier_connection_id),
    updated_at = NOW()
FROM carrier_connections AS cc
WHERE pn.id = sqlc.arg(id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.provisioning_mode = 'byoc'
  AND pn.status = 'active'
  AND cc.id = sqlc.arg(carrier_connection_id)
  AND cc.scope = 'organization'
  AND cc.organization_id = pn.organization_id
  AND cc.status = 'active'
RETURNING pn.*;

-- name: SetManagedPhoneNumberCarrierConnection :one
UPDATE phone_numbers AS pn
SET
    carrier_connection_id = sqlc.arg(carrier_connection_id),
    updated_at = NOW()
FROM carrier_connections AS cc
WHERE pn.id = sqlc.arg(id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.provisioning_mode = 'managed'
  AND pn.status = 'active'
  AND pn.provider_id IS NOT NULL
  AND cc.id = sqlc.arg(carrier_connection_id)
  AND cc.scope = 'platform'
  AND cc.organization_id IS NULL
  AND cc.provider_id = pn.provider_id
  AND cc.status = 'active'
RETURNING pn.*;

-- name: DisableManagedPhoneNumberForRelease :one
UPDATE phone_numbers
SET
    status = 'disabled',
    voice_enabled = false,
    sms_enabled = false,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND provisioning_mode = 'managed'
  AND status IN ('active', 'disabled')
RETURNING *;

-- name: ReleaseManagedPhoneNumber :one
UPDATE phone_numbers
SET
    status = 'released',
    carrier_connection_id = NULL,
    voice_enabled = false,
    sms_enabled = false,
    updated_at = now()
WHERE id = sqlc.arg(id)
  AND provisioning_mode = 'managed'
  AND status IN ('active', 'disabled')
RETURNING *;

-- name: ReleaseBYOCPhoneNumber :one
UPDATE phone_numbers
SET
    status = 'released',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND provisioning_mode = 'byoc'
  AND status IN ('active', 'disabled')
RETURNING *;

-- name: GetVoiceBindingByNumber :one
SELECT
    vb.id AS binding_id,
    vb.voice_application_id,
    va.name AS application_name,
    va.ring_timeout_seconds,
    va.caller_id AS application_caller_id,
    pn.id AS phone_number_id,
    pn.number,
    pn.organization_id
FROM phone_numbers AS pn
JOIN voice_bindings AS vb ON vb.phone_number_id = pn.id
JOIN voice_applications AS va ON va.id = vb.voice_application_id
JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.number = sqlc.arg(number)
  AND pn.status = 'active'
  AND pn.voice_enabled = true
  AND va.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListBackofficePhoneNumbers :many
SELECT
    pn.id::TEXT AS id,
    pn.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    pn.number,
    pn.country_code::TEXT AS country_code,
    pn.provisioning_mode,
    COALESCE(cp.name, 'BYOC') AS provider_name,
    pn.voice_enabled,
    pn.sms_enabled,
    pn.status,
    to_char(pn.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM phone_numbers AS pn
JOIN organizations AS o ON o.id = pn.organization_id
LEFT JOIN carrier_providers AS cp ON cp.id = pn.provider_id
ORDER BY pn.created_at DESC
LIMIT 100;

-- name: GetBackofficePhoneNumber :one
SELECT
    pn.id::TEXT AS id,
    pn.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    pn.number,
    pn.country_code::TEXT AS country_code,
    pn.provisioning_mode,
    CAST(COALESCE(pn.carrier_connection_id::TEXT, '—') AS TEXT) AS carrier_connection_id,
    COALESCE(cc.name, '—') AS carrier_connection_name,
    CAST(COALESCE(pn.provider_id::TEXT, '—') AS TEXT) AS provider_id,
    COALESCE(cp.name, 'BYOC') AS provider_name,
    COALESCE(pn.provider_resource_id, '—') AS provider_resource_id,
    pn.voice_enabled,
    pn.sms_enabled,
    pn.status,
    COALESCE(pn.error_code, '—') AS error_code,
    COALESCE(pn.error_message, '—') AS error_message,
    to_char(pn.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
    to_char(pn.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM phone_numbers AS pn
JOIN organizations AS o ON o.id = pn.organization_id
LEFT JOIN carrier_connections AS cc ON cc.id = pn.carrier_connection_id
LEFT JOIN carrier_providers AS cp ON cp.id = pn.provider_id
WHERE pn.id = sqlc.arg(id)
LIMIT 1;

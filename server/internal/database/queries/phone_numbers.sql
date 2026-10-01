-- name: CreateBYOCPhoneNumber :one
INSERT INTO phone_numbers (organization_id, number, country_code, carrier_connection_id, voice_enabled)
SELECT
    sqlc.arg(organization_id) AS organization_id,
    sqlc.arg(number) AS number,
    sqlc.arg(country_code) AS country_code,
    sqlc.narg(carrier_connection_id) AS carrier_connection_id,
    COALESCE(sqlc.narg(voice_enabled), true) AS voice_enabled
FROM organizations AS o
LEFT JOIN carrier_connections AS cc
  ON cc.id = sqlc.narg(carrier_connection_id)::UUID
 AND cc.organization_id = sqlc.arg(organization_id)
 AND cc.status = 'active'
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active' AND o.deleted_at IS NULL
  AND (sqlc.narg(carrier_connection_id)::UUID IS NULL OR cc.id IS NOT NULL)
RETURNING *;

-- name: GetPhoneNumberByID :one
SELECT pn.* FROM phone_numbers AS pn JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.id = sqlc.arg(id) AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status <> 'released' AND o.status = 'active' AND o.deleted_at IS NULL LIMIT 1;

-- name: GetPhoneNumberByNumber :one
SELECT pn.* FROM phone_numbers AS pn JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.number = sqlc.arg(number) AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.status = 'active' AND o.status = 'active' AND o.deleted_at IS NULL LIMIT 1;

-- name: ListPhoneNumbersByOrganizationID :many
SELECT * FROM phone_numbers WHERE organization_id = sqlc.arg(organization_id) AND status <> 'released' ORDER BY created_at DESC;

-- name: ListPhoneNumbersByCountry :many
SELECT * FROM phone_numbers WHERE organization_id = sqlc.arg(organization_id) AND country_code = sqlc.arg(country_code) AND status <> 'released' ORDER BY number ASC;

-- name: UpdatePhoneNumber :one
UPDATE phone_numbers SET voice_enabled = COALESCE(sqlc.narg(voice_enabled), voice_enabled),
    updated_at = NOW()
WHERE id = sqlc.arg(id) AND organization_id = sqlc.arg(organization_id) AND status = 'active' RETURNING *;

-- name: SetBYOCPhoneNumberCarrierConnection :one
UPDATE phone_numbers AS pn SET carrier_connection_id = sqlc.arg(carrier_connection_id), updated_at = NOW()
FROM carrier_connections AS cc
WHERE pn.id = sqlc.arg(id) AND pn.organization_id = sqlc.arg(organization_id) AND pn.status = 'active'
  AND cc.id = sqlc.arg(carrier_connection_id) AND cc.organization_id = pn.organization_id AND cc.status = 'active'
RETURNING pn.*;

-- name: ReleaseBYOCPhoneNumber :one
UPDATE phone_numbers SET status = 'released', carrier_connection_id = NULL, voice_enabled = false, updated_at = NOW()
WHERE id = sqlc.arg(id) AND organization_id = sqlc.arg(organization_id) AND status IN ('active', 'disabled') RETURNING *;

-- name: GetVoiceAgentBindingByNumber :one
SELECT
    binding.id AS binding_id,
    binding.voice_agent_id,
    pn.id AS phone_number_id,
    pn.number,
    pn.organization_id
FROM phone_numbers AS pn
JOIN voice_agent_bindings AS binding
  ON binding.phone_number_id = pn.id
 AND binding.organization_id = pn.organization_id
JOIN voice_agents AS agent
  ON agent.id = binding.voice_agent_id
 AND agent.organization_id = binding.organization_id
JOIN organizations AS o ON o.id = pn.organization_id
WHERE pn.number = sqlc.arg(number)
  AND pn.status = 'active'
  AND pn.voice_enabled = true
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListBackofficePhoneNumbers :many
SELECT pn.id::TEXT AS id, pn.organization_id::TEXT AS organization_id, o.name AS organization_name,
       pn.number, pn.country_code::TEXT AS country_code, COALESCE(cc.name, '—') AS carrier_connection_name,
       pn.voice_enabled, pn.status,
       to_char(pn.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM phone_numbers AS pn JOIN organizations AS o ON o.id = pn.organization_id
LEFT JOIN carrier_connections AS cc ON cc.id = pn.carrier_connection_id
ORDER BY pn.created_at DESC LIMIT 100;

-- name: GetBackofficePhoneNumber :one
SELECT pn.id::TEXT AS id, pn.organization_id::TEXT AS organization_id, o.name AS organization_name,
       pn.number, pn.country_code::TEXT AS country_code,
       CAST(COALESCE(pn.carrier_connection_id::TEXT, '—') AS TEXT) AS carrier_connection_id,
       COALESCE(cc.name, '—') AS carrier_connection_name, pn.voice_enabled, pn.status,
       to_char(pn.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at,
       to_char(pn.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS updated_at
FROM phone_numbers AS pn JOIN organizations AS o ON o.id = pn.organization_id
LEFT JOIN carrier_connections AS cc ON cc.id = pn.carrier_connection_id
WHERE pn.id = sqlc.arg(id) LIMIT 1;

-- name: CreateCall :one
INSERT INTO calls (
    organization_id,
    application_id,
    direction,
    state,
    from_uri,
    to_uri,
    sip_call_id
) VALUES (
    sqlc.arg(organization_id),
    sqlc.narg(application_id),
    sqlc.arg(direction),
    COALESCE(sqlc.narg(state), 'initiating'),
    sqlc.arg(from_uri),
    sqlc.arg(to_uri),
    sqlc.narg(sip_call_id)
)
RETURNING *;

-- name: GetCall :one
SELECT *
FROM calls
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetCallBySIPCallID :one
SELECT *
FROM calls
WHERE organization_id = sqlc.arg(organization_id)
  AND sip_call_id = sqlc.arg(sip_call_id)
LIMIT 1;

-- name: GetCallBySIPCallIDGlobal :one
SELECT *
FROM calls
WHERE sip_call_id = sqlc.arg(sip_call_id)
LIMIT 1;

-- name: ListCalls :many
SELECT *
FROM calls
WHERE organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(state)::text IS NULL OR state = sqlc.narg(state)::text)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: ListCallsForReconciliation :many
SELECT *
FROM calls
WHERE state IN ('initiating', 'ringing', 'answered', 'active')
  AND sip_call_id IS NOT NULL
  AND updated_at <= sqlc.arg(updated_before)
ORDER BY updated_at ASC
LIMIT sqlc.arg(batch_size);

-- Revalidate the DID-derived tenant and route tuple before the call service
-- persists or admits an inbound call.
-- name: GetInboundCallContext :one
SELECT
    cc.max_cps,
    cc.max_concurrent_calls,
    cc.max_daily_minutes
FROM phone_numbers AS pn
JOIN carrier_connections AS cc
  ON cc.id = pn.carrier_connection_id
JOIN voice_bindings AS vb
  ON vb.phone_number_id = pn.id
JOIN voice_applications AS va
  ON va.id = vb.voice_application_id
JOIN organizations AS o
  ON o.id = pn.organization_id
WHERE pn.id = sqlc.arg(phone_number_id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.number = sqlc.arg(called_number)
  AND pn.carrier_connection_id = sqlc.arg(carrier_connection_id)
  AND pn.status = 'active'
  AND pn.voice_enabled = true
  AND cc.status = 'active'
  AND cc.inbound_enabled = true
  AND (
      (
          cc.scope = 'organization'
          AND cc.organization_id = pn.organization_id
          AND pn.provisioning_mode = 'byoc'
      )
      OR (
          cc.scope = 'platform'
          AND cc.organization_id IS NULL
          AND pn.provisioning_mode = 'managed'
      )
  )
  AND va.id = sqlc.arg(application_id)
  AND va.organization_id = pn.organization_id
  AND va.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: SetCallRouteAttribution :one
UPDATE calls
SET
    carrier_connection_id = sqlc.arg(carrier_connection_id),
    trunk_id = sqlc.arg(trunk_id),
    trunk_endpoint_id = sqlc.arg(trunk_endpoint_id),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: UpdateCallState :one
UPDATE calls
SET
    state = sqlc.arg(state),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: MarkCallRinging :one
UPDATE calls
SET state = 'ringing', updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state = 'initiating'
RETURNING *;

-- name: MarkCallAnswered :one
UPDATE calls
SET
    state = 'answered',
    answered_at = COALESCE(answered_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('initiating', 'ringing')
RETURNING *;

-- name: MarkCallActive :one
UPDATE calls
SET state = 'active', updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('answered', 'ringing')
RETURNING *;

-- name: MarkCallHeld :one
UPDATE calls
SET
    media_state = 'held',
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('answered', 'active')
  AND media_state = 'active'
RETURNING *;

-- name: MarkCallResumed :one
UPDATE calls
SET
    media_state = 'active',
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('answered', 'active')
  AND media_state = 'held'
RETURNING *;

-- name: MarkCallCompleted :one
UPDATE calls
SET
    state = 'completed',
    ended_at = COALESCE(ended_at, NOW()),
    hangup_reason = COALESCE(sqlc.narg(hangup_reason), hangup_reason),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state NOT IN ('completed', 'failed', 'cancelled')
RETURNING *;

-- name: MarkCallFailed :one
UPDATE calls
SET
    state = 'failed',
    ended_at = COALESCE(ended_at, NOW()),
    hangup_reason = COALESCE(sqlc.narg(hangup_reason), hangup_reason),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state NOT IN ('completed', 'failed', 'cancelled')
RETURNING *;

-- name: MarkCallCancelled :one
UPDATE calls
SET
    state = 'cancelled',
    ended_at = COALESCE(ended_at, NOW()),
    hangup_reason = COALESCE(sqlc.narg(hangup_reason), hangup_reason),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('initiating', 'ringing')
RETURNING *;

-- name: ListBackofficeCalls :many
SELECT
    c.id::TEXT AS id,
    c.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    c.from_uri,
    c.to_uri,
    c.direction,
    c.state,
    CAST(
        GREATEST(
            0::BIGINT,
            COALESCE(
                EXTRACT(EPOCH FROM (COALESCE(c.ended_at, NOW()) - c.answered_at))::BIGINT,
                0::BIGINT
            )
        ) AS BIGINT
    ) AS duration_seconds,
    to_char(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI') AS created_at
FROM calls AS c
JOIN organizations AS o ON o.id = c.organization_id
ORDER BY c.created_at DESC
LIMIT 100;

-- name: GetBackofficeCall :one
SELECT
    c.id::TEXT AS id,
    c.organization_id::TEXT AS organization_id,
    o.name AS organization_name,
    c.direction,
    c.state,
    c.media_state,
    c.from_uri,
    c.to_uri,
    COALESCE(c.sip_call_id, '—')::TEXT AS sip_call_id,
    COALESCE(c.application_id::TEXT, '—')::TEXT AS application_id,
    COALESCE(va.name, '—')::TEXT AS application_name,
    COALESCE(c.carrier_connection_id::TEXT, '—')::TEXT AS carrier_connection_id,
    COALESCE(cc.name, '—')::TEXT AS carrier_connection_name,
    COALESCE(cp.id::TEXT, '—')::TEXT AS provider_id,
    COALESCE(cp.name, '—')::TEXT AS provider_name,
    COALESCE(c.trunk_id::TEXT, '—')::TEXT AS trunk_id,
    COALESCE(t.name, '—')::TEXT AS trunk_name,
    COALESCE(c.trunk_endpoint_id::TEXT, '—')::TEXT AS trunk_endpoint_id,
    COALESCE(c.hangup_reason, '—')::TEXT AS hangup_reason,
    CAST(
        GREATEST(
            0::BIGINT,
            COALESCE(
                EXTRACT(EPOCH FROM (COALESCE(c.ended_at, NOW()) - c.answered_at))::BIGINT,
                0::BIGINT
            )
        ) AS BIGINT
    ) AS duration_seconds,
    COUNT(DISTINCT r.id)::BIGINT AS recording_count,
    COALESCE(string_agg(DISTINCT r.status, ', ' ORDER BY r.status), 'none')::TEXT AS recording_status,
    COALESCE(to_char(c.started_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), '—')::TEXT AS started_at,
    COALESCE(to_char(c.answered_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), '—')::TEXT AS answered_at,
    COALESCE(to_char(c.ended_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS'), '—')::TEXT AS ended_at,
    to_char(c.created_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')::TEXT AS created_at,
    to_char(c.updated_at AT TIME ZONE 'UTC', 'YYYY-MM-DD HH24:MI:SS')::TEXT AS updated_at
FROM calls AS c
JOIN organizations AS o ON o.id = c.organization_id
LEFT JOIN voice_applications AS va ON va.id = c.application_id
LEFT JOIN carrier_connections AS cc ON cc.id = c.carrier_connection_id
LEFT JOIN carrier_providers AS cp ON cp.id = cc.provider_id
LEFT JOIN trunks AS t ON t.id = c.trunk_id
LEFT JOIN recordings AS r
  ON r.call_id = c.id
 AND r.organization_id = c.organization_id
WHERE c.id = sqlc.arg(id)
GROUP BY c.id, o.name, va.name, cc.name, cp.id, cp.name, t.name
LIMIT 1;

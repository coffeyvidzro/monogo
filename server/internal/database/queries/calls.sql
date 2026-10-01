-- name: CreateCall :one
INSERT INTO calls (
    organization_id,
    voice_agent_id,
    direction,
    state,
    from_uri,
    to_uri,
    sip_call_id
) VALUES (
    sqlc.arg(organization_id),
    sqlc.narg(voice_agent_id),
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

-- name: GetCallLifecycleSnapshot :one
SELECT
    c.organization_id,
    c.trunk_id,
    c.direction,
    c.state,
    c.media_state,
    c.answered_at
FROM calls AS c
WHERE c.id = sqlc.arg(id)
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

-- name: ListActiveCallsForAdmissionReconciliation :many
SELECT
    id,
    trunk_id
FROM calls
WHERE state IN ('initiating', 'ringing', 'answered', 'active')
  AND trunk_id IS NOT NULL
ORDER BY created_at ASC;

-- Revalidate the DID-derived tenant and route tuple before the call service
-- persists or admits an inbound call.
-- name: GetInboundCallContext :one
SELECT
    t.max_cps,
    t.max_concurrent_calls
FROM phone_numbers AS pn
JOIN trunks AS t
  ON t.id = pn.trunk_id
JOIN voice_agent_bindings AS binding
  ON binding.phone_number_id = pn.id
 AND binding.organization_id = pn.organization_id
JOIN voice_agents AS agent
  ON agent.id = binding.voice_agent_id
 AND agent.organization_id = binding.organization_id
JOIN organizations AS o
  ON o.id = pn.organization_id
WHERE pn.id = sqlc.arg(phone_number_id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.number = sqlc.arg(called_number)
  AND pn.trunk_id = sqlc.arg(trunk_id)
  AND pn.status = 'active'
  AND pn.voice_enabled = true
  AND t.status = 'active'
  AND t.inbound_enabled = true
  AND t.organization_id = pn.organization_id
  AND binding.id = sqlc.arg(voice_agent_binding_id)
  AND agent.id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: SetCallRouteAttribution :one
UPDATE calls
SET
    trunk_id = sqlc.arg(trunk_id),
    trunk_endpoint_id = sqlc.arg(trunk_endpoint_id),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: SetOutboundCallSIPCallID :one
-- Bind the actual FreeSWITCH SIP dialog identity after asynchronous originate.
-- Replayed events with the same identity are idempotent; a different Call-ID
-- must never overwrite a previously attributed dialog.
UPDATE calls
SET
    sip_call_id = sqlc.arg(sip_call_id),
    updated_at = CASE WHEN sip_call_id IS NULL THEN NOW() ELSE updated_at END
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND (sip_call_id IS NULL OR sip_call_id = sqlc.arg(sip_call_id))
RETURNING id;

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
    COALESCE(c.voice_agent_id::TEXT, '—')::TEXT AS voice_agent_id,
    COALESCE(agent.name, '—')::TEXT AS voice_agent_name,
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
LEFT JOIN voice_agents AS agent ON agent.id = c.voice_agent_id
LEFT JOIN trunks AS t ON t.id = c.trunk_id
LEFT JOIN recordings AS r
  ON r.call_id = c.id
 AND r.organization_id = c.organization_id
WHERE c.id = sqlc.arg(id)
GROUP BY c.id, o.name, agent.name, t.name, t.name
LIMIT 1;

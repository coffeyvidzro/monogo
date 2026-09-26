-- name: CreateVoiceAgentTool :one
INSERT INTO voice_agent_tools (
    organization_id,
    voice_agent_id,
    type,
    name,
    description,
    parameters,
    endpoint_url,
    timeout_ms,
    enabled
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(voice_agent_id),
    sqlc.arg(type),
    sqlc.arg(name),
    sqlc.arg(description),
    sqlc.arg(parameters),
    sqlc.narg(endpoint_url),
    sqlc.arg(timeout_ms),
    sqlc.arg(enabled)
FROM voice_agents AS agent
WHERE agent.id = sqlc.arg(voice_agent_id)
  AND agent.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
RETURNING *;

-- name: ListVoiceAgentToolsByAgentID :many
SELECT tool.*
FROM voice_agent_tools AS tool
JOIN voice_agents AS agent ON agent.id = tool.voice_agent_id
WHERE tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
ORDER BY tool.created_at ASC;

-- name: GetVoiceAgentToolByID :one
SELECT tool.*
FROM voice_agent_tools AS tool
JOIN voice_agents AS agent ON agent.id = tool.voice_agent_id
WHERE tool.id = sqlc.arg(id)
  AND tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
LIMIT 1;

-- name: UpdateVoiceAgentTool :one
UPDATE voice_agent_tools
SET
    name = COALESCE(sqlc.narg(name), name),
    description = COALESCE(sqlc.narg(description), description),
    parameters = COALESCE(sqlc.narg(parameters), parameters),
    endpoint_url = COALESCE(sqlc.narg(endpoint_url), endpoint_url),
    timeout_ms = COALESCE(sqlc.narg(timeout_ms), timeout_ms),
    enabled = COALESCE(sqlc.narg(enabled), enabled),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id)
RETURNING *;

-- name: DeleteVoiceAgentTool :exec
DELETE FROM voice_agent_tools
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id);

-- name: CreateVoiceAgentSession :one
INSERT INTO voice_agent_sessions (
    organization_id,
    call_id,
    voice_agent_id,
    engine,
    instructions_snapshot,
    voice,
    language
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(call_id),
    agent.id,
    agent.engine,
    agent.instructions,
    agent.voice,
    agent.language
FROM voice_agents AS agent
JOIN calls AS c
  ON c.id = sqlc.arg(call_id)
 AND c.organization_id = sqlc.arg(organization_id)
WHERE agent.id = sqlc.arg(voice_agent_id)
  AND agent.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
RETURNING *;

-- name: GetVoiceAgentSessionByID :one
SELECT *
FROM voice_agent_sessions
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetActiveVoiceAgentSessionByCallID :one
SELECT *
FROM voice_agent_sessions
WHERE organization_id = sqlc.arg(organization_id)
  AND call_id = sqlc.arg(call_id)
  AND state = 'active'
LIMIT 1;

-- name: CompleteVoiceAgentSession :one
UPDATE voice_agent_sessions
SET
    state = sqlc.arg(state),
    turn_count = sqlc.arg(turn_count),
    interruption_count = sqlc.arg(interruption_count),
    first_response_latency_ms = sqlc.narg(first_response_latency_ms),
    avg_turn_latency_ms = sqlc.narg(avg_turn_latency_ms),
    ended_at = sqlc.arg(ended_at),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND state = 'active'
RETURNING *;

-- name: CreateVoiceAgentTurn :one
INSERT INTO voice_agent_turns (
    organization_id,
    session_id,
    sequence,
    role,
    content,
    provider_id,
    tool_name,
    tool_call_id,
    metadata,
    speech_started_at,
    speech_ended_at,
    stt_latency_ms,
    llm_ttft_ms,
    tts_ttfb_ms,
    turn_latency_ms
)
SELECT
    sqlc.arg(organization_id),
    session.id,
    sqlc.arg(sequence),
    sqlc.arg(role),
    sqlc.arg(content),
    sqlc.narg(provider_id),
    sqlc.narg(tool_name),
    sqlc.narg(tool_call_id),
    sqlc.arg(metadata),
    sqlc.narg(speech_started_at),
    sqlc.narg(speech_ended_at),
    sqlc.narg(stt_latency_ms),
    sqlc.narg(llm_ttft_ms),
    sqlc.narg(tts_ttfb_ms),
    sqlc.narg(turn_latency_ms)
FROM voice_agent_sessions AS session
WHERE session.id = sqlc.arg(session_id)
  AND session.organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: ListVoiceAgentTurnsBySessionID :many
SELECT *
FROM voice_agent_turns
WHERE organization_id = sqlc.arg(organization_id)
  AND session_id = sqlc.arg(session_id)
ORDER BY sequence ASC;

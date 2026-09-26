-- name: CreateVoiceAgentSession :one
INSERT INTO voice_agent_sessions (
    organization_id,
    call_id,
    voice_agent_id,
    engine,
    instructions_snapshot,
    engine_config_snapshot,
    voice,
    language
)
SELECT
    sqlc.arg(organization_id),
    c.id,
    agent.id,
    agent.engine,
    agent.instructions,
    agent.engine_config,
    agent.voice,
    agent.language
FROM calls AS c
JOIN organizations AS o
  ON o.id = c.organization_id
JOIN voice_applications AS app
  ON app.id = c.application_id
 AND app.organization_id = c.organization_id
JOIN voice_agent_bindings AS binding
  ON binding.voice_application_id = app.id
 AND binding.organization_id = app.organization_id
JOIN voice_agents AS agent
  ON agent.id = binding.voice_agent_id
 AND agent.organization_id = binding.organization_id
WHERE c.id = sqlc.arg(call_id)
  AND c.organization_id = sqlc.arg(organization_id)
  AND agent.id = sqlc.arg(voice_agent_id)
  AND c.state IN ('answered', 'active')
  AND c.ended_at IS NULL
  AND app.status = 'active'
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: GetVoiceAgentSessionByID :one
SELECT session.*
FROM voice_agent_sessions AS session
WHERE session.id = sqlc.arg(id)
  AND session.organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetActiveVoiceAgentSessionByCallID :one
SELECT session.*
FROM voice_agent_sessions AS session
JOIN organizations AS o ON o.id = session.organization_id
WHERE session.organization_id = sqlc.arg(organization_id)
  AND session.call_id = sqlc.arg(call_id)
  AND session.state = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: CompleteVoiceAgentSession :one
UPDATE voice_agent_sessions AS session
SET
    state = sqlc.arg(state),
    turn_count = sqlc.arg(turn_count),
    interruption_count = sqlc.arg(interruption_count),
    first_response_latency_ms = sqlc.narg(first_response_latency_ms),
    avg_turn_latency_ms = sqlc.narg(avg_turn_latency_ms),
    ended_at = sqlc.arg(ended_at),
    updated_at = NOW()
WHERE session.id = sqlc.arg(id)
  AND session.organization_id = sqlc.arg(organization_id)
  AND session.state = 'active'
RETURNING session.*;

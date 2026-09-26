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

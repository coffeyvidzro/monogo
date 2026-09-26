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
JOIN organizations AS o ON o.id = session.organization_id
WHERE session.id = sqlc.arg(session_id)
  AND session.organization_id = sqlc.arg(organization_id)
  AND session.state = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: ListVoiceAgentTurnsBySessionID :many
SELECT turn.*
FROM voice_agent_turns AS turn
JOIN voice_agent_sessions AS session
  ON session.id = turn.session_id
 AND session.organization_id = turn.organization_id
WHERE turn.organization_id = sqlc.arg(organization_id)
  AND turn.session_id = sqlc.arg(session_id)
ORDER BY turn.sequence ASC;

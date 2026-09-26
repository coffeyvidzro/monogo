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

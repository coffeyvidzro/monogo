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
    agent.id,
    sqlc.arg(type),
    sqlc.arg(name),
    sqlc.arg(description),
    sqlc.arg(parameters),
    sqlc.narg(endpoint_url),
    sqlc.arg(timeout_ms),
    sqlc.arg(enabled)
FROM voice_agents AS agent
JOIN organizations AS o ON o.id = agent.organization_id
WHERE agent.id = sqlc.arg(voice_agent_id)
  AND agent.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: ListVoiceAgentToolsByAgentID :many
SELECT tool.*
FROM voice_agent_tools AS tool
JOIN voice_agents AS agent
  ON agent.id = tool.voice_agent_id
 AND agent.organization_id = tool.organization_id
JOIN organizations AS o ON o.id = tool.organization_id
WHERE tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY tool.created_at ASC;

-- name: GetVoiceAgentToolByID :one
SELECT tool.*
FROM voice_agent_tools AS tool
JOIN voice_agents AS agent
  ON agent.id = tool.voice_agent_id
 AND agent.organization_id = tool.organization_id
JOIN organizations AS o ON o.id = tool.organization_id
WHERE tool.id = sqlc.arg(id)
  AND tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: UpdateVoiceAgentTool :one
UPDATE voice_agent_tools AS tool
SET
    name = COALESCE(sqlc.narg(name), tool.name),
    description = COALESCE(sqlc.narg(description), tool.description),
    parameters = COALESCE(sqlc.narg(parameters), tool.parameters),
    endpoint_url = COALESCE(sqlc.narg(endpoint_url), tool.endpoint_url),
    timeout_ms = COALESCE(sqlc.narg(timeout_ms), tool.timeout_ms),
    enabled = COALESCE(sqlc.narg(enabled), tool.enabled),
    updated_at = NOW()
FROM voice_agents AS agent, organizations AS o
WHERE tool.id = sqlc.arg(id)
  AND tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.id = tool.voice_agent_id
  AND agent.organization_id = tool.organization_id
  AND agent.status = 'active'
  AND o.id = tool.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING tool.*;

-- name: DeleteVoiceAgentTool :exec
DELETE FROM voice_agent_tools AS tool
USING voice_agents AS agent, organizations AS o
WHERE tool.id = sqlc.arg(id)
  AND tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.id = tool.voice_agent_id
  AND agent.organization_id = tool.organization_id
  AND agent.status = 'active'
  AND o.id = tool.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

-- name: CreateVoiceAgentBinding :one
INSERT INTO voice_agent_bindings (
    organization_id,
    voice_agent_id,
    phone_number_id
)
SELECT
    sqlc.arg(organization_id),
    agent.id,
    sqlc.arg(phone_number_id)::UUID
FROM voice_agents AS agent
JOIN organizations AS o ON o.id = agent.organization_id
WHERE agent.id = sqlc.arg(voice_agent_id)
  AND agent.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: GetVoiceAgentBindingByID :one
SELECT binding.*
FROM voice_agent_bindings AS binding
JOIN voice_agents AS agent
  ON agent.id = binding.voice_agent_id
 AND agent.organization_id = binding.organization_id
JOIN organizations AS o ON o.id = binding.organization_id
WHERE binding.id = sqlc.arg(id)
  AND binding.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListVoiceAgentBindingsByAgentID :many
SELECT binding.*
FROM voice_agent_bindings AS binding
JOIN voice_agents AS agent
  ON agent.id = binding.voice_agent_id
 AND agent.organization_id = binding.organization_id
WHERE binding.organization_id = sqlc.arg(organization_id)
  AND binding.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
ORDER BY binding.created_at DESC;

-- name: DeleteVoiceAgentBinding :exec
DELETE FROM voice_agent_bindings
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id);

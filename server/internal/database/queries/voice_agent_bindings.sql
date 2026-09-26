-- name: CreateVoiceAgentBinding :one
INSERT INTO voice_agent_bindings (
    organization_id,
    voice_agent_id,
    voice_application_id
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(voice_agent_id),
    sqlc.arg(voice_application_id)
FROM voice_agents AS agent
JOIN voice_applications AS app
  ON app.id = sqlc.arg(voice_application_id)
 AND app.organization_id = sqlc.arg(organization_id)
JOIN organizations AS o
  ON o.id = sqlc.arg(organization_id)
WHERE agent.id = sqlc.arg(voice_agent_id)
  AND agent.organization_id = sqlc.arg(organization_id)
  AND agent.status = 'active'
  AND app.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: ListVoiceAgentBindingsByAgentID :many
SELECT vab.*
FROM voice_agent_bindings AS vab
JOIN voice_agents AS agent ON agent.id = vab.voice_agent_id
WHERE vab.organization_id = sqlc.arg(organization_id)
  AND vab.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
ORDER BY vab.created_at DESC;

-- name: DeleteVoiceAgentBinding :exec
DELETE FROM voice_agent_bindings
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id);

-- name: GetVoiceAgentByApplicationID :one
SELECT agent.*
FROM voice_agent_bindings AS vab
JOIN voice_agents AS agent
  ON agent.id = vab.voice_agent_id
 AND agent.organization_id = vab.organization_id
JOIN voice_applications AS app
  ON app.id = vab.voice_application_id
 AND app.organization_id = vab.organization_id
WHERE vab.organization_id = sqlc.arg(organization_id)
  AND vab.voice_application_id = sqlc.arg(voice_application_id)
  AND agent.status = 'active'
  AND app.status = 'active'
LIMIT 1;

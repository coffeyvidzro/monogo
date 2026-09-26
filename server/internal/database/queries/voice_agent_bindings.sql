-- name: CreateVoiceAgentBinding :one
INSERT INTO voice_agent_bindings (
    organization_id,
    voice_agent_id,
    voice_application_id
)
SELECT
    sqlc.arg(organization_id),
    agent.id,
    app.id
FROM organizations AS o
JOIN voice_agents AS agent
  ON agent.organization_id = o.id
 AND agent.id = sqlc.arg(voice_agent_id)
JOIN voice_applications AS app
  ON app.organization_id = o.id
 AND app.id = sqlc.arg(voice_application_id)
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
  AND agent.status = 'active'
  AND app.status = 'active'
RETURNING *;

-- name: ListVoiceAgentBindingsByAgentID :many
SELECT vab.*
FROM voice_agent_bindings AS vab
JOIN voice_agents AS agent
  ON agent.id = vab.voice_agent_id
 AND agent.organization_id = vab.organization_id
JOIN voice_applications AS app
  ON app.id = vab.voice_application_id
 AND app.organization_id = vab.organization_id
JOIN organizations AS o
  ON o.id = vab.organization_id
WHERE vab.organization_id = sqlc.arg(organization_id)
  AND vab.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.status = 'active'
  AND app.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY vab.created_at DESC;

-- name: DeleteVoiceAgentBinding :exec
DELETE FROM voice_agent_bindings AS vab
USING voice_agents AS agent, voice_applications AS app, organizations AS o
WHERE vab.id = sqlc.arg(id)
  AND vab.organization_id = sqlc.arg(organization_id)
  AND vab.voice_agent_id = sqlc.arg(voice_agent_id)
  AND agent.id = vab.voice_agent_id
  AND agent.organization_id = vab.organization_id
  AND agent.status = 'active'
  AND app.id = vab.voice_application_id
  AND app.organization_id = vab.organization_id
  AND app.status = 'active'
  AND o.id = vab.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

-- name: GetVoiceAgentByApplicationID :one
SELECT agent.*
FROM voice_agent_bindings AS vab
JOIN voice_agents AS agent
  ON agent.id = vab.voice_agent_id
 AND agent.organization_id = vab.organization_id
JOIN voice_applications AS app
  ON app.id = vab.voice_application_id
 AND app.organization_id = vab.organization_id
JOIN organizations AS o
  ON o.id = vab.organization_id
WHERE vab.organization_id = sqlc.arg(organization_id)
  AND vab.voice_application_id = sqlc.arg(voice_application_id)
  AND agent.status = 'active'
  AND app.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

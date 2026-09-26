-- name: CreateVoiceAgent :one
INSERT INTO voice_agents (
    organization_id,
    name,
    engine,
    instructions,
    voice,
    language
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.arg(engine),
    sqlc.arg(instructions),
    sqlc.narg(voice),
    sqlc.narg(language)
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: GetVoiceAgentByID :one
SELECT va.*
FROM voice_agents AS va
JOIN organizations AS o ON o.id = va.organization_id
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: ListVoiceAgentsByOrganizationID :many
SELECT va.*
FROM voice_agents AS va
JOIN organizations AS o ON o.id = va.organization_id
WHERE va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY va.created_at DESC;

-- name: UpdateVoiceAgent :one
UPDATE voice_agents
SET
    name = COALESCE(sqlc.narg(name), name),
    engine = COALESCE(sqlc.narg(engine), engine),
    instructions = COALESCE(sqlc.narg(instructions), instructions),
    voice = COALESCE(sqlc.narg(voice), voice),
    language = COALESCE(sqlc.narg(language), language),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active'
RETURNING *;

-- name: DisableVoiceAgent :exec
UPDATE voice_agents
SET
    status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND status = 'active';

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

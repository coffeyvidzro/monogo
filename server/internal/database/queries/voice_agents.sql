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
UPDATE voice_agents AS va
SET
    name = COALESCE(sqlc.narg(name), va.name),
    engine = COALESCE(sqlc.narg(engine), va.engine),
    instructions = COALESCE(sqlc.narg(instructions), va.instructions),
    voice = COALESCE(sqlc.narg(voice), va.voice),
    language = COALESCE(sqlc.narg(language), va.language),
    updated_at = NOW()
FROM organizations AS o
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND o.id = va.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING va.*;

-- name: DisableVoiceAgent :exec
UPDATE voice_agents AS va
SET
    status = 'disabled',
    updated_at = NOW()
FROM organizations AS o
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND o.id = va.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

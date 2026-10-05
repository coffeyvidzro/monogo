-- name: CreateVoiceAgent :one
INSERT INTO voice_agents (
    organization_id,
    name,
    engine,
    instructions,
    voice,
    language,
    engine_config,
    preset,
    preset_version,
    interruption_policy,
    recording_policy
)
SELECT
    sqlc.arg(organization_id),
    sqlc.arg(name),
    sqlc.arg(engine),
    sqlc.arg(instructions),
    sqlc.narg(voice),
    sqlc.narg(language),
    COALESCE(sqlc.narg(engine_config)::jsonb, '{}'::jsonb),
    sqlc.narg(preset),
    sqlc.narg(preset_version),
    sqlc.arg(interruption_policy),
    sqlc.arg(recording_policy)
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
    engine_config = COALESCE(sqlc.narg(engine_config)::jsonb, va.engine_config),
    preset = CASE WHEN sqlc.arg(update_preset)::boolean THEN sqlc.narg(preset) ELSE va.preset END,
    preset_version = CASE WHEN sqlc.arg(update_preset)::boolean THEN sqlc.narg(preset_version) ELSE va.preset_version END,
    interruption_policy = COALESCE(sqlc.narg(interruption_policy), va.interruption_policy),
    recording_policy = COALESCE(sqlc.narg(recording_policy), va.recording_policy),
    configuration_revision = va.configuration_revision + 1,
    updated_at = NOW()
FROM organizations AS o
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND o.id = va.organization_id
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING va.*;

-- name: ActivateVoiceAgent :one
UPDATE voice_agents AS va
SET active_revision = va.configuration_revision,
    active_engine = va.engine,
    active_instructions = va.instructions,
    active_voice = va.voice,
    active_language = va.language,
    active_engine_config = va.engine_config,
    active_interruption_policy = va.interruption_policy,
    active_recording_policy = va.recording_policy,
    active_provider_bindings = COALESCE((
        SELECT jsonb_agg(
            jsonb_build_object(
                'role', binding.role,
                'provider', binding.provider,
                'credential_id', binding.credential_id,
                'config', binding.config
            )
            ORDER BY binding.role
        )
        FROM voice_agent_provider_bindings AS binding
        WHERE binding.organization_id = va.organization_id
          AND binding.voice_agent_id = va.id
    ), '[]'::jsonb),
    active_tools = COALESCE((
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', tool.id,
                'name', tool.name,
                'description', tool.description,
                'parameters', tool.parameters,
                'type', tool.type,
                'endpoint_url', tool.endpoint_url,
                'timeout_ms', tool.timeout_ms
            )
            ORDER BY tool.name
        )
        FROM voice_agent_tools AS tool
        WHERE tool.organization_id = va.organization_id
          AND tool.voice_agent_id = va.id
          AND tool.enabled = TRUE
    ), '[]'::jsonb),
    updated_at = NOW()
WHERE va.id = sqlc.arg(id)
  AND va.organization_id = sqlc.arg(organization_id)
  AND va.status = 'active'
  AND NOT EXISTS (
      SELECT 1
      FROM voice_agent_provider_bindings AS binding
      JOIN ai_provider_credentials AS integration
        ON integration.id = binding.credential_id
       AND integration.organization_id = binding.organization_id
      WHERE binding.organization_id = va.organization_id
        AND binding.voice_agent_id = va.id
        AND integration.connection_state <> 'ready'
  )
  AND (
      (va.engine = 'integrated' AND (
          SELECT count(*)
          FROM voice_agent_provider_bindings AS binding
          WHERE binding.organization_id = va.organization_id
            AND binding.voice_agent_id = va.id
            AND binding.role = 'realtime'
            AND binding.provider = 'openai'
      ) = 1 AND (
          SELECT count(*)
          FROM voice_agent_provider_bindings AS binding
          WHERE binding.organization_id = va.organization_id
            AND binding.voice_agent_id = va.id
      ) = 1)
      OR
      (va.engine = 'composable' AND (
          SELECT count(*)
          FROM voice_agent_provider_bindings AS binding
          WHERE binding.organization_id = va.organization_id
            AND binding.voice_agent_id = va.id
            AND (
                (binding.role = 'stt' AND binding.provider = 'deepgram') OR
                (binding.role = 'llm' AND binding.provider = 'groq') OR
                (binding.role = 'tts' AND binding.provider = 'cartesia')
            )
      ) = 3 AND (
          SELECT count(*)
          FROM voice_agent_provider_bindings AS binding
          WHERE binding.organization_id = va.organization_id
            AND binding.voice_agent_id = va.id
      ) = 3)
  )
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

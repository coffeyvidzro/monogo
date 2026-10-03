-- name: CreateAIProviderCredential :one
INSERT INTO ai_provider_credentials (
    id, organization_id, provider, name, secret_ciphertext
) VALUES (
    sqlc.arg(id), sqlc.arg(organization_id), sqlc.arg(provider),
    sqlc.arg(name), sqlc.arg(secret_ciphertext)
)
RETURNING *;

-- name: ListAIProviderCredentials :many
SELECT *
FROM ai_provider_credentials
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY provider ASC, name ASC, created_at ASC;

-- name: GetAIProviderCredential :one
SELECT *
FROM ai_provider_credentials
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);

-- name: RotateAIProviderCredential :one
UPDATE ai_provider_credentials
SET secret_ciphertext = sqlc.arg(secret_ciphertext),
    rotated_at = now(),
    connection_state = 'unchecked',
    verified_at = NULL,
    failure_code = NULL
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: UpdateAIProviderCredentialVerification :one
UPDATE ai_provider_credentials
SET connection_state = sqlc.arg(connection_state),
    verified_at = now(),
    failure_code = sqlc.narg(failure_code)
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: ListVoiceAgentIDsByAIProviderCredential :many
SELECT voice_agent_id
FROM voice_agent_provider_bindings
WHERE organization_id = sqlc.arg(organization_id)
  AND credential_id = sqlc.arg(credential_id)
ORDER BY voice_agent_id;

-- name: DeleteAIProviderCredential :exec
DELETE FROM ai_provider_credentials
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id);

-- name: UpsertVoiceAgentProviderBinding :one
INSERT INTO voice_agent_provider_bindings (
    organization_id, voice_agent_id, role, provider, credential_id, config
) VALUES (
    sqlc.arg(organization_id), sqlc.arg(voice_agent_id), sqlc.arg(role),
    sqlc.arg(provider), sqlc.arg(credential_id), sqlc.arg(config)
)
ON CONFLICT (organization_id, voice_agent_id, role)
DO UPDATE SET
    provider = EXCLUDED.provider,
    credential_id = EXCLUDED.credential_id,
    config = EXCLUDED.config
RETURNING *;

-- name: ListVoiceAgentProviderBindings :many
SELECT *
FROM voice_agent_provider_bindings
WHERE organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id)
ORDER BY role ASC;

-- name: DeleteVoiceAgentProviderBinding :exec
DELETE FROM voice_agent_provider_bindings
WHERE organization_id = sqlc.arg(organization_id)
  AND voice_agent_id = sqlc.arg(voice_agent_id)
  AND role = sqlc.arg(role);

-- name: ResolveVoiceAgentProviderBindings :many
SELECT
    b.id,
    b.organization_id,
    b.voice_agent_id,
    b.role,
    b.provider,
    b.credential_id,
    b.config,
    c.secret_ciphertext,
    b.created_at,
    b.updated_at
FROM voice_agent_provider_bindings b
JOIN ai_provider_credentials c
  ON c.id = b.credential_id
 AND c.organization_id = b.organization_id
WHERE b.organization_id = sqlc.arg(organization_id)
  AND b.voice_agent_id = sqlc.arg(voice_agent_id)
ORDER BY b.role ASC;

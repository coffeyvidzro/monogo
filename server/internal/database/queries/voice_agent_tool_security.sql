-- name: CreateVoiceAgentWebhookTool :one
WITH inserted_tool AS (
    INSERT INTO voice_agent_tools (
        id,
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
        sqlc.arg(id),
        sqlc.arg(organization_id),
        agent.id,
        'webhook',
        sqlc.arg(name),
        sqlc.arg(description),
        sqlc.arg(parameters)::jsonb,
        sqlc.arg(endpoint_url),
        sqlc.arg(timeout_ms),
        sqlc.arg(enabled)
    FROM voice_agents AS agent
    JOIN organizations AS o ON o.id = agent.organization_id
    WHERE agent.id = sqlc.arg(voice_agent_id)
      AND agent.organization_id = sqlc.arg(organization_id)
      AND agent.status = 'active'
      AND o.status = 'active'
      AND o.deleted_at IS NULL
    RETURNING id, organization_id, voice_agent_id
),
inserted_secret AS (
    INSERT INTO voice_agent_tool_secrets (
        tool_id,
        organization_id,
        voice_agent_id,
        secret_ciphertext
    )
    SELECT
        tool.id,
        tool.organization_id,
        tool.voice_agent_id,
        sqlc.arg(secret_ciphertext)
    FROM inserted_tool AS tool
    RETURNING tool_id
)
SELECT tool_id
FROM inserted_secret;

-- name: GetVoiceAgentToolSigningSecret :one
SELECT secret.secret_ciphertext
FROM voice_agent_tool_secrets AS secret
JOIN voice_agent_tools AS tool
  ON tool.id = secret.tool_id
 AND tool.organization_id = secret.organization_id
 AND tool.voice_agent_id = secret.voice_agent_id
JOIN voice_agents AS agent
  ON agent.id = tool.voice_agent_id
 AND agent.organization_id = tool.organization_id
JOIN organizations AS o ON o.id = tool.organization_id
WHERE secret.tool_id = sqlc.arg(tool_id)
  AND secret.organization_id = sqlc.arg(organization_id)
  AND secret.voice_agent_id = sqlc.arg(voice_agent_id)
  AND tool.type = 'webhook'
  AND tool.enabled = true
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
LIMIT 1;

-- name: RotateVoiceAgentToolSigningSecret :execrows
INSERT INTO voice_agent_tool_secrets (
    tool_id,
    organization_id,
    voice_agent_id,
    secret_ciphertext
)
SELECT
    tool.id,
    tool.organization_id,
    tool.voice_agent_id,
    sqlc.arg(secret_ciphertext)
FROM voice_agent_tools AS tool
JOIN voice_agents AS agent
  ON agent.id = tool.voice_agent_id
 AND agent.organization_id = tool.organization_id
JOIN organizations AS o ON o.id = tool.organization_id
WHERE tool.id = sqlc.arg(tool_id)
  AND tool.organization_id = sqlc.arg(organization_id)
  AND tool.voice_agent_id = sqlc.arg(voice_agent_id)
  AND tool.type = 'webhook'
  AND agent.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (tool_id) DO UPDATE
SET
    secret_ciphertext = EXCLUDED.secret_ciphertext,
    rotated_at = NOW();

-- name: ClaimVoiceAgentToolExecution :one
INSERT INTO voice_agent_tool_executions (
    organization_id,
    session_id,
    voice_agent_id,
    tool_id,
    call_id,
    tool_call_id,
    arguments
)
SELECT
    session.organization_id,
    session.id,
    session.voice_agent_id,
    tool.id,
    session.call_id,
    sqlc.arg(tool_call_id),
    sqlc.arg(arguments)::jsonb
FROM voice_agent_sessions AS session
JOIN voice_agent_tools AS tool
  ON tool.id = sqlc.arg(tool_id)
 AND tool.organization_id = session.organization_id
 AND tool.voice_agent_id = session.voice_agent_id
JOIN organizations AS o ON o.id = session.organization_id
WHERE session.id = sqlc.arg(session_id)
  AND session.organization_id = sqlc.arg(organization_id)
  AND session.voice_agent_id = sqlc.arg(voice_agent_id)
  AND session.call_id = sqlc.arg(call_id)
  AND session.state = 'active'
  AND tool.enabled = true
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ON CONFLICT (organization_id, session_id, tool_call_id) DO NOTHING
RETURNING id;

-- name: GetVoiceAgentToolExecution :one
SELECT execution.*
FROM voice_agent_tool_executions AS execution
WHERE execution.organization_id = sqlc.arg(organization_id)
  AND execution.session_id = sqlc.arg(session_id)
  AND execution.tool_call_id = sqlc.arg(tool_call_id)
LIMIT 1;

-- name: MarkVoiceAgentToolExecutionSucceeded :execrows
UPDATE voice_agent_tool_executions
SET
    state = 'succeeded',
    response_status = sqlc.narg(response_status),
    response_content_type = sqlc.narg(response_content_type),
    response_body = sqlc.narg(response_body),
    error_message = NULL,
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND state = 'processing';

-- name: MarkVoiceAgentToolExecutionFailed :execrows
UPDATE voice_agent_tool_executions
SET
    state = 'failed',
    error_message = sqlc.arg(error_message),
    completed_at = NOW(),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
  AND state = 'processing';

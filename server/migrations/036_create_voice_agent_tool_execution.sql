CREATE TABLE IF NOT EXISTS voice_agent_tool_secrets (
    tool_id UUID PRIMARY KEY REFERENCES voice_agent_tools(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,
    secret_ciphertext TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_tool_secrets_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,

    CONSTRAINT chk_voice_agent_tool_secrets_ciphertext
        CHECK (length(btrim(secret_ciphertext)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_tool_secrets_scope
    ON voice_agent_tool_secrets (organization_id, voice_agent_id);

CREATE TABLE IF NOT EXISTS voice_agent_tool_executions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    session_id UUID NOT NULL REFERENCES voice_agent_sessions(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,
    tool_id UUID NOT NULL REFERENCES voice_agent_tools(id) ON DELETE CASCADE,
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,

    tool_call_id TEXT NOT NULL,
    state TEXT NOT NULL DEFAULT 'processing',
    arguments JSONB NOT NULL DEFAULT '{}'::jsonb,

    response_status INTEGER,
    response_content_type TEXT,
    response_body BYTEA,
    error_message TEXT,

    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_tool_executions_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,

    CONSTRAINT chk_voice_agent_tool_executions_call_id
        CHECK (call_id <> '00000000-0000-0000-0000-000000000000'::uuid),

    CONSTRAINT chk_voice_agent_tool_executions_tool_call_id
        CHECK (length(btrim(tool_call_id)) BETWEEN 1 AND 255),

    CONSTRAINT chk_voice_agent_tool_executions_state
        CHECK (state IN ('processing', 'succeeded', 'failed')),

    CONSTRAINT chk_voice_agent_tool_executions_arguments
        CHECK (jsonb_typeof(arguments) = 'object'),

    CONSTRAINT chk_voice_agent_tool_executions_response_status
        CHECK (response_status IS NULL OR response_status BETWEEN 100 AND 599),

    CONSTRAINT chk_voice_agent_tool_executions_lifecycle
        CHECK (
            (state = 'processing' AND completed_at IS NULL AND error_message IS NULL)
            OR
            (state = 'succeeded' AND completed_at IS NOT NULL AND error_message IS NULL)
            OR
            (state = 'failed' AND completed_at IS NOT NULL AND error_message IS NOT NULL)
        ),

    CONSTRAINT uq_voice_agent_tool_executions_call
        UNIQUE (organization_id, session_id, tool_call_id)
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_tool_executions_session
    ON voice_agent_tool_executions (organization_id, session_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_agent_tool_executions_tool
    ON voice_agent_tool_executions (organization_id, tool_id, created_at DESC);

CREATE TRIGGER set_voice_agent_tool_executions_updated_at
BEFORE UPDATE ON voice_agent_tool_executions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

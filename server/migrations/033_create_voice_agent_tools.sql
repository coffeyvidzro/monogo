CREATE TABLE IF NOT EXISTS voice_agent_tools (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,

    type TEXT NOT NULL DEFAULT 'webhook',
    name TEXT NOT NULL,
    description TEXT NOT NULL,
    parameters JSONB NOT NULL DEFAULT '{}'::jsonb,
    endpoint_url TEXT,
    timeout_ms INTEGER NOT NULL DEFAULT 3000,
    enabled BOOLEAN NOT NULL DEFAULT true,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Constraints & Scoping
    CONSTRAINT uq_voice_agent_tools_agent_name UNIQUE (voice_agent_id, name),
    
    CONSTRAINT fk_voice_agent_tools_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,

    -- Validation Rules
    CONSTRAINT chk_voice_agent_tools_type CHECK (type IN ('builtin', 'webhook')),
    CONSTRAINT chk_voice_agent_tools_name CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_voice_agent_tools_description CHECK (length(btrim(description)) BETWEEN 1 AND 2000),
    CONSTRAINT chk_voice_agent_tools_parameters CHECK (jsonb_typeof(parameters) = 'object'),
    CONSTRAINT chk_voice_agent_tools_endpoint CHECK (
        (type = 'builtin' AND endpoint_url IS NULL)
        OR
        (type = 'webhook' AND endpoint_url ~ '^https?://')
    ),
    CONSTRAINT chk_voice_agent_tools_timeout CHECK (timeout_ms BETWEEN 100 AND 30000)
);

-- Optimized Index for Runtime Tool Resolution
CREATE INDEX IF NOT EXISTS idx_voice_agent_tools_agent_enabled
    ON voice_agent_tools (organization_id, voice_agent_id, enabled);

-- Trigger for Updated At
CREATE TRIGGER set_voice_agent_tools_updated_at
BEFORE UPDATE ON voice_agent_tools
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

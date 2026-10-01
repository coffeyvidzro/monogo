CREATE TABLE IF NOT EXISTS ai_provider_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    provider TEXT NOT NULL,
    name TEXT NOT NULL,
    secret_ciphertext TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    rotated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_ai_provider_credentials_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_ai_provider_credentials_name UNIQUE (organization_id, provider, name),
    CONSTRAINT chk_ai_provider_credentials_provider
        CHECK (provider IN ('openai', 'deepgram', 'groq', 'cartesia')),
    CONSTRAINT chk_ai_provider_credentials_name
        CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_ai_provider_credentials_secret
        CHECK (length(btrim(secret_ciphertext)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_ai_provider_credentials_organization
    ON ai_provider_credentials (organization_id, provider, created_at DESC);

CREATE TRIGGER set_ai_provider_credentials_updated_at
BEFORE UPDATE ON ai_provider_credentials
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS voice_agent_provider_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,
    role TEXT NOT NULL,
    provider TEXT NOT NULL,
    credential_id UUID NOT NULL,
    config JSONB NOT NULL DEFAULT '{}'::jsonb,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_provider_bindings_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_provider_bindings_credential_scope
        FOREIGN KEY (credential_id, organization_id)
        REFERENCES ai_provider_credentials(id, organization_id)
        ON DELETE RESTRICT,

    CONSTRAINT uq_voice_agent_provider_bindings_role
        UNIQUE (organization_id, voice_agent_id, role),
    CONSTRAINT chk_voice_agent_provider_bindings_role
        CHECK (role IN ('realtime', 'stt', 'llm', 'tts')),
    CONSTRAINT chk_voice_agent_provider_bindings_provider
        CHECK (provider IN ('openai', 'deepgram', 'groq', 'cartesia')),
    CONSTRAINT chk_voice_agent_provider_bindings_config
        CHECK (jsonb_typeof(config) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_provider_bindings_agent
    ON voice_agent_provider_bindings (organization_id, voice_agent_id);

CREATE TRIGGER set_voice_agent_provider_bindings_updated_at
BEFORE UPDATE ON voice_agent_provider_bindings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

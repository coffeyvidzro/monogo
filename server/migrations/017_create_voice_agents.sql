CREATE TABLE IF NOT EXISTS voice_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name TEXT NOT NULL,
    engine TEXT NOT NULL,
    instructions TEXT NOT NULL,
    voice TEXT,
    language TEXT,
    status TEXT NOT NULL DEFAULT 'active',
    engine_config JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_voice_agents_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_voice_agents_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_voice_agents_name CHECK (length(btrim(name)) BETWEEN 1 AND 255),
    CONSTRAINT chk_voice_agents_engine CHECK (engine IN ('composable', 'integrated')),
    CONSTRAINT chk_voice_agents_instructions CHECK (length(btrim(instructions)) BETWEEN 1 AND 20000),
    CONSTRAINT chk_voice_agents_voice CHECK (voice IS NULL OR length(btrim(voice)) BETWEEN 1 AND 255),
    CONSTRAINT chk_voice_agents_language CHECK (language IS NULL OR length(btrim(language)) BETWEEN 1 AND 64),
    CONSTRAINT chk_voice_agents_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_voice_agents_engine_config CHECK (jsonb_typeof(engine_config) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_voice_agents_organization_status
    ON voice_agents (organization_id, status);

CREATE TABLE IF NOT EXISTS voice_agent_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,
    phone_number_id UUID,
    sip_domain_id UUID,
    subscriber_id UUID,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_bindings_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_bindings_phone_number_scope
        FOREIGN KEY (phone_number_id, organization_id)
        REFERENCES phone_numbers(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_bindings_sip_domain_scope
        FOREIGN KEY (sip_domain_id, organization_id)
        REFERENCES sip_domains(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_bindings_subscriber_scope
        FOREIGN KEY (subscriber_id, organization_id)
        REFERENCES subscribers(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT chk_voice_agent_bindings_single_target CHECK (
        (phone_number_id IS NOT NULL)::int
        + (sip_domain_id IS NOT NULL)::int
        + (subscriber_id IS NOT NULL)::int = 1
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_bindings_phone_number
    ON voice_agent_bindings (phone_number_id)
    WHERE phone_number_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_bindings_sip_domain
    ON voice_agent_bindings (sip_domain_id)
    WHERE sip_domain_id IS NOT NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_bindings_subscriber
    ON voice_agent_bindings (subscriber_id)
    WHERE subscriber_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_voice_agent_bindings_agent
    ON voice_agent_bindings (organization_id, voice_agent_id, created_at DESC);

CREATE TRIGGER set_voice_agents_updated_at
BEFORE UPDATE ON voice_agents
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

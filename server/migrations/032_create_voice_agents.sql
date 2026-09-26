CREATE TABLE IF NOT EXISTS voice_agents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    engine TEXT NOT NULL,
    instructions TEXT NOT NULL,
    voice TEXT,
    language TEXT,
    status TEXT NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_voice_agents_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_voice_agents_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_voice_agents_name CHECK (length(btrim(name)) BETWEEN 1 AND 255),
    CONSTRAINT chk_voice_agents_engine CHECK (engine IN ('composable', 'integrated')),
    CONSTRAINT chk_voice_agents_instructions CHECK (length(btrim(instructions)) BETWEEN 1 AND 20000),
    CONSTRAINT chk_voice_agents_voice CHECK (voice IS NULL OR length(btrim(voice)) BETWEEN 1 AND 255),
    CONSTRAINT chk_voice_agents_language CHECK (language IS NULL OR length(btrim(language)) BETWEEN 1 AND 64),
    CONSTRAINT chk_voice_agents_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_voice_agents_organization_status
    ON voice_agents (organization_id, status);

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_applications_id_organization
    ON voice_applications (id, organization_id);

CREATE TABLE IF NOT EXISTS voice_agent_bindings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    voice_agent_id UUID NOT NULL,
    voice_application_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_bindings_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_bindings_application_scope
        FOREIGN KEY (voice_application_id, organization_id)
        REFERENCES voice_applications(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_voice_agent_bindings_application UNIQUE (voice_application_id),
    CONSTRAINT uq_voice_agent_bindings_agent_application UNIQUE (voice_agent_id, voice_application_id)
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_bindings_agent
    ON voice_agent_bindings (organization_id, voice_agent_id);

CREATE TRIGGER set_voice_agents_updated_at
BEFORE UPDATE ON voice_agents
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

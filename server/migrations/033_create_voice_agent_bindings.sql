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

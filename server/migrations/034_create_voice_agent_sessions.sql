CREATE TABLE IF NOT EXISTS voice_agent_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    call_id UUID NOT NULL,
    voice_agent_id UUID NOT NULL,

    engine TEXT NOT NULL,
    instructions_snapshot TEXT NOT NULL,
    engine_config_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    voice TEXT,
    language TEXT,
    state TEXT NOT NULL DEFAULT 'active',

    turn_count INTEGER NOT NULL DEFAULT 0,
    interruption_count INTEGER NOT NULL DEFAULT 0,
    first_response_latency_ms INTEGER,
    avg_turn_latency_ms INTEGER,

    started_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- Constraints & Multi-Tenant Scoping
    CONSTRAINT uq_voice_agent_sessions_id_organization UNIQUE (id, organization_id),
    
    CONSTRAINT fk_voice_agent_sessions_call_scope
        FOREIGN KEY (call_id, organization_id)
        REFERENCES calls(id, organization_id)
        ON DELETE CASCADE,

    CONSTRAINT fk_voice_agent_sessions_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE RESTRICT,

    -- Data Validation Constraints
    CONSTRAINT chk_voice_agent_sessions_engine CHECK (engine IN ('composable', 'integrated')),
    CONSTRAINT chk_voice_agent_sessions_instructions CHECK (length(btrim(instructions_snapshot)) BETWEEN 1 AND 20000),
    CONSTRAINT chk_voice_agent_sessions_voice CHECK (voice IS NULL OR length(btrim(voice)) BETWEEN 1 AND 255),
    CONSTRAINT chk_voice_agent_sessions_language CHECK (language IS NULL OR length(btrim(language)) BETWEEN 1 AND 64),
    CONSTRAINT chk_voice_agent_sessions_state CHECK (state IN ('active', 'completed', 'failed', 'cancelled')),
    CONSTRAINT chk_voice_agent_sessions_turn_count CHECK (turn_count >= 0),
    CONSTRAINT chk_voice_agent_sessions_interruption_count CHECK (interruption_count >= 0),
    CONSTRAINT chk_voice_agent_sessions_first_response_latency CHECK (
        first_response_latency_ms IS NULL OR first_response_latency_ms >= 0
    ),
    CONSTRAINT chk_voice_agent_sessions_avg_turn_latency CHECK (
        avg_turn_latency_ms IS NULL OR avg_turn_latency_ms >= 0
    ),
    CONSTRAINT chk_voice_agent_sessions_timestamps CHECK (
        ended_at IS NULL OR ended_at >= started_at
    ),
    CONSTRAINT chk_voice_agent_sessions_lifecycle CHECK (
        (state = 'active' AND ended_at IS NULL)
        OR
        (state <> 'active' AND ended_at IS NOT NULL)
    )
);

-- Partial Index: Enforce One Active Session Per Call
CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_sessions_active_call
    ON voice_agent_sessions (call_id)
    WHERE state = 'active';

-- High-Frequency Query Indexes
CREATE INDEX IF NOT EXISTS idx_voice_agent_sessions_organization_agent
    ON voice_agent_sessions (organization_id, voice_agent_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_agent_sessions_call
    ON voice_agent_sessions (call_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_agent_sessions_call_scope
    ON voice_agent_sessions (call_id, organization_id);

-- Trigger for Updated At
CREATE TRIGGER set_voice_agent_sessions_updated_at
BEFORE UPDATE ON voice_agent_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

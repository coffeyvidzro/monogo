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

    CONSTRAINT fk_voice_agent_tools_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_voice_agent_tools_agent_name UNIQUE (voice_agent_id, name),
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

CREATE INDEX IF NOT EXISTS idx_voice_agent_tools_agent_enabled
    ON voice_agent_tools (organization_id, voice_agent_id, enabled);

CREATE TABLE IF NOT EXISTS voice_agent_sessions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    call_id UUID NOT NULL,
    voice_agent_id UUID NOT NULL,

    engine TEXT NOT NULL,
    instructions_snapshot TEXT NOT NULL,
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

    CONSTRAINT uq_voice_agent_sessions_id_organization UNIQUE (id, organization_id),
    CONSTRAINT fk_voice_agent_sessions_call_scope
        FOREIGN KEY (call_id, organization_id)
        REFERENCES calls(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_voice_agent_sessions_agent_scope
        FOREIGN KEY (voice_agent_id, organization_id)
        REFERENCES voice_agents(id, organization_id)
        ON DELETE RESTRICT,
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

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_sessions_active_call
    ON voice_agent_sessions (call_id)
    WHERE state = 'active';

CREATE INDEX IF NOT EXISTS idx_voice_agent_sessions_organization_agent
    ON voice_agent_sessions (organization_id, voice_agent_id, started_at DESC);

CREATE INDEX IF NOT EXISTS idx_voice_agent_sessions_call
    ON voice_agent_sessions (call_id, started_at DESC);

CREATE TABLE IF NOT EXISTS voice_agent_turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,

    sequence INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL,
    provider_id TEXT,
    tool_name TEXT,
    tool_call_id TEXT,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,

    speech_started_at TIMESTAMPTZ,
    speech_ended_at TIMESTAMPTZ,
    stt_latency_ms INTEGER,
    llm_ttft_ms INTEGER,
    tts_ttfb_ms INTEGER,
    turn_latency_ms INTEGER,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_turns_session_scope
        FOREIGN KEY (session_id, organization_id)
        REFERENCES voice_agent_sessions(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_voice_agent_turns_session_sequence UNIQUE (session_id, sequence),
    CONSTRAINT chk_voice_agent_turns_sequence CHECK (sequence > 0),
    CONSTRAINT chk_voice_agent_turns_role CHECK (role IN ('user', 'assistant', 'tool', 'system')),
    CONSTRAINT chk_voice_agent_turns_content CHECK (length(btrim(content)) > 0),
    CONSTRAINT chk_voice_agent_turns_metadata CHECK (jsonb_typeof(metadata) = 'object'),
    CONSTRAINT chk_voice_agent_turns_speech_timestamps CHECK (
        speech_ended_at IS NULL
        OR speech_started_at IS NULL
        OR speech_ended_at >= speech_started_at
    ),
    CONSTRAINT chk_voice_agent_turns_stt_latency CHECK (stt_latency_ms IS NULL OR stt_latency_ms >= 0),
    CONSTRAINT chk_voice_agent_turns_llm_ttft CHECK (llm_ttft_ms IS NULL OR llm_ttft_ms >= 0),
    CONSTRAINT chk_voice_agent_turns_tts_ttfb CHECK (tts_ttfb_ms IS NULL OR tts_ttfb_ms >= 0),
    CONSTRAINT chk_voice_agent_turns_turn_latency CHECK (turn_latency_ms IS NULL OR turn_latency_ms >= 0)
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_turns_session_created
    ON voice_agent_turns (session_id, sequence);

CREATE INDEX IF NOT EXISTS idx_voice_agent_turns_organization_created
    ON voice_agent_turns (organization_id, created_at DESC);

CREATE TRIGGER set_voice_agent_tools_updated_at
BEFORE UPDATE ON voice_agent_tools
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TRIGGER set_voice_agent_sessions_updated_at
BEFORE UPDATE ON voice_agent_sessions
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

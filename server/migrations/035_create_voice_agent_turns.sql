CREATE TABLE IF NOT EXISTS voice_agent_turns (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,

    sequence INTEGER NOT NULL,
    role TEXT NOT NULL,
    content TEXT NOT NULL DEFAULT '',
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
    CONSTRAINT chk_voice_agent_turns_content_or_tool CHECK (
        length(btrim(content)) > 0 OR tool_call_id IS NOT NULL
    ),
    
    CONSTRAINT chk_voice_agent_turns_tool_roles CHECK (
        (role = 'tool' AND tool_call_id IS NOT NULL) OR (role <> 'tool')
    ),

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

-- Optimize for analytics and fetching recent turns per organization
CREATE INDEX IF NOT EXISTS idx_voice_agent_turns_organization_created
    ON voice_agent_turns (organization_id, created_at DESC);

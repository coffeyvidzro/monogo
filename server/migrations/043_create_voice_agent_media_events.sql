CREATE TABLE IF NOT EXISTS voice_agent_media_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    session_id UUID NOT NULL,
    generation BIGINT NOT NULL,
    event_type TEXT NOT NULL,
    provider_id TEXT,
    tool_name TEXT,
    text_content TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    processing_started_at TIMESTAMPTZ,
    processed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_voice_agent_media_events_session_scope
        FOREIGN KEY (session_id, organization_id)
        REFERENCES voice_agent_sessions(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_voice_agent_media_events_generation
        UNIQUE (session_id, generation),
    CONSTRAINT chk_voice_agent_media_events_generation
        CHECK (generation > 0),
    CONSTRAINT chk_voice_agent_media_events_type
        CHECK (event_type IN (
            'speech.started',
            'interrupted',
            'transcript.final',
            'response.stopped',
            'tool.call',
            'usage',
            'error'
        )),
    CONSTRAINT chk_voice_agent_media_events_payload
        CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX IF NOT EXISTS idx_voice_agent_media_events_session
    ON voice_agent_media_events (organization_id, session_id, generation);

CREATE INDEX IF NOT EXISTS idx_voice_agent_media_events_pending
    ON voice_agent_media_events (created_at)
    WHERE processed_at IS NULL;

CREATE UNIQUE INDEX IF NOT EXISTS uq_voice_agent_media_events_provider_event
    ON voice_agent_media_events (session_id, event_type, provider_id)
    WHERE provider_id IS NOT NULL;

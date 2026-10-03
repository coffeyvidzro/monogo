ALTER TABLE voice_agents
    ADD COLUMN preset TEXT,
    ADD COLUMN preset_version INTEGER,
    ADD COLUMN interruption_policy TEXT NOT NULL DEFAULT 'allow',
    ADD COLUMN recording_policy TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN configuration_revision INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN active_revision INTEGER,
    ADD COLUMN active_engine TEXT,
    ADD COLUMN active_instructions TEXT,
    ADD COLUMN active_voice TEXT,
    ADD COLUMN active_language TEXT,
    ADD COLUMN active_engine_config JSONB,
    ADD COLUMN active_interruption_policy TEXT,
    ADD COLUMN active_recording_policy TEXT,
    ADD COLUMN active_provider_bindings JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN active_tools JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT chk_voice_agents_preset
        CHECK (preset IS NULL OR preset IN ('balanced', 'fast', 'high_intelligence', 'realtime')),
    ADD CONSTRAINT chk_voice_agents_preset_version
        CHECK ((preset IS NULL AND preset_version IS NULL) OR (preset IS NOT NULL AND preset_version > 0)),
    ADD CONSTRAINT chk_voice_agents_interruption_policy
        CHECK (interruption_policy IN ('allow', 'disabled')),
    ADD CONSTRAINT chk_voice_agents_recording_policy
        CHECK (recording_policy IN ('none', 'all')),
    ADD CONSTRAINT chk_voice_agents_configuration_revision
        CHECK (configuration_revision > 0),
    ADD CONSTRAINT chk_voice_agents_active_revision
        CHECK (active_revision IS NULL OR active_revision > 0),
    ADD CONSTRAINT chk_voice_agents_active_provider_bindings
        CHECK (jsonb_typeof(active_provider_bindings) = 'array'),
    ADD CONSTRAINT chk_voice_agents_active_tools
        CHECK (jsonb_typeof(active_tools) = 'array');

UPDATE voice_agents
SET active_revision = configuration_revision,
    active_engine = engine,
    active_instructions = instructions,
    active_voice = voice,
    active_language = language,
    active_engine_config = engine_config,
    active_interruption_policy = interruption_policy,
    active_recording_policy = recording_policy,
    active_provider_bindings = COALESCE((
        SELECT jsonb_agg(
            jsonb_build_object(
                'role', binding.role,
                'provider', binding.provider,
                'credential_id', binding.credential_id,
                'config', binding.config
            )
            ORDER BY binding.role
        )
        FROM voice_agent_provider_bindings AS binding
        WHERE binding.organization_id = voice_agents.organization_id
          AND binding.voice_agent_id = voice_agents.id
    ), '[]'::jsonb),
    active_tools = COALESCE((
        SELECT jsonb_agg(
            jsonb_build_object(
                'id', tool.id,
                'name', tool.name,
                'description', tool.description,
                'parameters', tool.parameters
            )
            ORDER BY tool.name
        )
        FROM voice_agent_tools AS tool
        WHERE tool.organization_id = voice_agents.organization_id
          AND tool.voice_agent_id = voice_agents.id
          AND tool.enabled = TRUE
    ), '[]'::jsonb);

ALTER TABLE voice_agent_sessions
    ADD COLUMN configuration_revision INTEGER NOT NULL DEFAULT 1,
    ADD COLUMN interruption_policy TEXT NOT NULL DEFAULT 'allow',
    ADD COLUMN recording_policy TEXT NOT NULL DEFAULT 'none',
    ADD COLUMN provider_bindings_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD COLUMN tools_snapshot JSONB NOT NULL DEFAULT '[]'::jsonb,
    ADD CONSTRAINT chk_voice_agent_sessions_configuration_revision
        CHECK (configuration_revision > 0),
    ADD CONSTRAINT chk_voice_agent_sessions_interruption_policy
        CHECK (interruption_policy IN ('allow', 'disabled')),
    ADD CONSTRAINT chk_voice_agent_sessions_recording_policy
        CHECK (recording_policy IN ('none', 'all')),
    ADD CONSTRAINT chk_voice_agent_sessions_provider_bindings_snapshot
        CHECK (jsonb_typeof(provider_bindings_snapshot) = 'array'),
    ADD CONSTRAINT chk_voice_agent_sessions_tools_snapshot
        CHECK (jsonb_typeof(tools_snapshot) = 'array');

CREATE FUNCTION bump_voice_agent_configuration_revision()
RETURNS TRIGGER AS $$
BEGIN
    UPDATE voice_agents
    SET configuration_revision = configuration_revision + 1,
        updated_at = now()
    WHERE id = COALESCE(NEW.voice_agent_id, OLD.voice_agent_id)
      AND organization_id = COALESCE(NEW.organization_id, OLD.organization_id);
    RETURN COALESCE(NEW, OLD);
END;
$$ LANGUAGE plpgsql;

CREATE TRIGGER bump_voice_agent_revision_on_provider_binding
AFTER INSERT OR UPDATE OR DELETE ON voice_agent_provider_bindings
FOR EACH ROW
EXECUTE FUNCTION bump_voice_agent_configuration_revision();

CREATE TRIGGER bump_voice_agent_revision_on_tool
AFTER INSERT OR UPDATE OR DELETE ON voice_agent_tools
FOR EACH ROW
EXECUTE FUNCTION bump_voice_agent_configuration_revision();

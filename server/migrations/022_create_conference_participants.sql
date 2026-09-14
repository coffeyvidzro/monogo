CREATE TABLE IF NOT EXISTS conference_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    conference_id UUID NOT NULL REFERENCES conferences(id) ON DELETE CASCADE,
    call_participant_id UUID REFERENCES call_participants(id) ON DELETE SET NULL,
    state TEXT NOT NULL DEFAULT 'joined',
    muted BOOLEAN NOT NULL DEFAULT false,
    deaf BOOLEAN NOT NULL DEFAULT false,
    speaking BOOLEAN NOT NULL DEFAULT false,
    joined_at TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_conference_participants_state CHECK (
        state IN ('joining', 'joined', 'left', 'failed')
    ),
    CONSTRAINT chk_conference_participants_left_at CHECK (
        left_at IS NULL OR joined_at IS NULL OR left_at >= joined_at
    )
);

CREATE INDEX IF NOT EXISTS idx_conference_participants_organization_id
    ON conference_participants (organization_id);

CREATE INDEX IF NOT EXISTS idx_conference_participants_conference_id
    ON conference_participants (conference_id, created_at);

CREATE INDEX IF NOT EXISTS idx_conference_participants_call_participant_id
    ON conference_participants (call_participant_id)
    WHERE call_participant_id IS NOT NULL;

CREATE TRIGGER set_conference_participants_updated_at
BEFORE UPDATE ON conference_participants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

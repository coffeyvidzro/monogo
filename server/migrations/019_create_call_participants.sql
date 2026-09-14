CREATE TABLE IF NOT EXISTS call_participants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    subscriber_id UUID REFERENCES subscribers(id) ON DELETE SET NULL,
    role TEXT NOT NULL,
    address TEXT,
    direction TEXT,
    state TEXT NOT NULL DEFAULT 'joined',
    joined_at TIMESTAMPTZ,
    left_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_call_participants_role CHECK (role IN ('caller', 'callee', 'bridge', 'other')),
    CONSTRAINT chk_call_participants_direction CHECK (
        direction IS NULL OR direction IN ('inbound', 'outbound')
    ),
    CONSTRAINT chk_call_participants_state CHECK (
        state IN ('joining', 'joined', 'left', 'failed')
    ),
    CONSTRAINT chk_call_participants_left_at CHECK (
        left_at IS NULL OR joined_at IS NULL OR left_at >= joined_at
    )
);

CREATE INDEX IF NOT EXISTS idx_call_participants_organization_id
    ON call_participants (organization_id);

CREATE INDEX IF NOT EXISTS idx_call_participants_call_id
    ON call_participants (call_id, created_at);

CREATE INDEX IF NOT EXISTS idx_call_participants_subscriber_id
    ON call_participants (subscriber_id)
    WHERE subscriber_id IS NOT NULL;

CREATE TRIGGER set_call_participants_updated_at
BEFORE UPDATE ON call_participants
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS conferences (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    application_id UUID REFERENCES voice_applications(id) ON DELETE SET NULL,
    name VARCHAR(255) NOT NULL,
    state TEXT NOT NULL DEFAULT 'active',
    started_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_conferences_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_conferences_name CHECK (length(trim(name)) > 0),
    CONSTRAINT chk_conferences_state CHECK (state IN ('active', 'ended')),
    CONSTRAINT chk_conferences_ended_at CHECK (
        ended_at IS NULL OR started_at IS NULL OR ended_at >= started_at
    )
);

CREATE INDEX IF NOT EXISTS idx_conferences_organization_created
    ON conferences (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_conferences_application_created
    ON conferences (application_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_conferences_organization_state
    ON conferences (organization_id, state, created_at DESC);

CREATE TRIGGER set_conferences_updated_at
BEFORE UPDATE ON conferences
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

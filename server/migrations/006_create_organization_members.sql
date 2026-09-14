CREATE TABLE IF NOT EXISTS organization_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role TEXT NOT NULL DEFAULT 'member',
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (organization_id, user_id),
    CONSTRAINT chk_organization_members_role CHECK (role IN ('owner', 'admin', 'member')),
    CONSTRAINT chk_organization_members_status CHECK (status IN ('active', 'disabled'))
);

-- Indexes for efficient querying of organization memberships
CREATE INDEX IF NOT EXISTS idx_organization_members_user_id
    ON organization_members (user_id);

CREATE INDEX IF NOT EXISTS idx_organization_members_organization_id_status
    ON organization_members (organization_id, status);

CREATE INDEX IF NOT EXISTS idx_organization_members_user_id_status
    ON organization_members (user_id, status);

-- Trigger to auto-update updated_at
CREATE TRIGGER set_organization_members_updated_at
BEFORE UPDATE ON organization_members
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

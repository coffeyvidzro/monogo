CREATE TABLE IF NOT EXISTS organization_invitations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    invited_by UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    email CITEXT NOT NULL,
    role TEXT NOT NULL DEFAULT 'member',
    token_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    accepted_at TIMESTAMPTZ,
    declined_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    -- One pending invitation per email per organization
    CONSTRAINT chk_organization_invitations_status
        CHECK (status IN ('pending', 'accepted', 'declined', 'expired', 'revoked')),
    CONSTRAINT chk_organization_invitations_role
        CHECK (role IN ('admin', 'member'))
);

-- Prevent duplicate pending invitations for same email + organization
CREATE UNIQUE INDEX idx_organization_invitations_pending_email
    ON organization_invitations (organization_id, email)
    WHERE status = 'pending';

CREATE INDEX idx_organization_invitations_organization_id
    ON organization_invitations (organization_id);

CREATE INDEX idx_organization_invitations_token_hash
    ON organization_invitations (token_hash);

CREATE INDEX idx_organization_invitations_email
    ON organization_invitations (email);

CREATE INDEX idx_organization_invitations_expires_at
    ON organization_invitations (expires_at)
    WHERE status = 'pending';

CREATE TRIGGER set_organization_invitations_updated_at
BEFORE UPDATE ON organization_invitations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

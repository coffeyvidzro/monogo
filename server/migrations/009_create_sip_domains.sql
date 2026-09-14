CREATE TABLE IF NOT EXISTS sip_domains (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    domain CITEXT UNIQUE NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_sip_domains_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_sip_domains_organization_id
    ON sip_domains (organization_id);

CREATE INDEX IF NOT EXISTS idx_sip_domains_status
    ON sip_domains (status);

-- Trigger to auto-update updated_at
CREATE TRIGGER set_sip_domains_updated_at
BEFORE UPDATE ON sip_domains
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

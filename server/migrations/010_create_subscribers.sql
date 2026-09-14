CREATE TABLE IF NOT EXISTS subscribers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    sip_domain_id UUID NOT NULL REFERENCES sip_domains(id) ON DELETE CASCADE,

    username VARCHAR(64) NOT NULL,
    domain CITEXT NOT NULL,

    ha1_md5 VARCHAR(255),
    ha1_sha256 VARCHAR(255),
    ha1_sha512_256 VARCHAR(255),
    display_name VARCHAR(255),
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (sip_domain_id, username),
    UNIQUE (domain, username)
);

CREATE INDEX idx_subscribers_organization_id ON subscribers (organization_id);
CREATE INDEX idx_subscribers_sip_domain_id ON subscribers (sip_domain_id);
CREATE INDEX idx_subscribers_status ON subscribers (status);
CREATE INDEX idx_subscribers_domain_username ON subscribers (domain, username);

CREATE TRIGGER set_subscribers_updated_at
BEFORE UPDATE ON subscribers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

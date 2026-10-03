CREATE TABLE IF NOT EXISTS sso_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    protocol TEXT NOT NULL,
    issuer TEXT NOT NULL,
    configuration JSONB NOT NULL DEFAULT '{}'::jsonb,
    secret_ciphertext TEXT,
    status TEXT NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_sso_connections_name UNIQUE (organization_id, name),
    CONSTRAINT chk_sso_connections_name
        CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_sso_connections_protocol
        CHECK (protocol IN ('saml', 'oidc')),
    CONSTRAINT chk_sso_connections_issuer
        CHECK (length(btrim(issuer)) BETWEEN 1 AND 2048),
    CONSTRAINT chk_sso_connections_configuration
        CHECK (jsonb_typeof(configuration) = 'object'),
    CONSTRAINT chk_sso_connections_status
        CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_sso_connections_organization
    ON sso_connections (organization_id, created_at DESC);

CREATE TRIGGER set_sso_connections_updated_at
BEFORE UPDATE ON sso_connections
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

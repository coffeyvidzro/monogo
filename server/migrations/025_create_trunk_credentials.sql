
CREATE TABLE trunk_credentials (
    trunk_id UUID PRIMARY KEY REFERENCES trunks(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    username TEXT NOT NULL,
    realm TEXT NOT NULL,
    ha1_md5 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_trunk_credentials_identity UNIQUE (username, realm),
    CONSTRAINT chk_trunk_credentials_username CHECK (
        length(btrim(username)) > 0
    ),
    CONSTRAINT chk_trunk_credentials_realm CHECK (
        length(btrim(realm)) > 0
    ),
    CONSTRAINT chk_trunk_credentials_ha1 CHECK (
        ha1_md5 ~ '^[0-9a-f]{32}$'
    ),
    CONSTRAINT fk_trunk_credentials_tenant FOREIGN KEY (trunk_id, organization_id)
        REFERENCES trunks(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_trunk_credentials_org
    ON trunk_credentials (organization_id);

CREATE TRIGGER set_trunk_credentials_updated_at
BEFORE UPDATE ON trunk_credentials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Stable auth_db contract for the hosted Leamout SIP edge. Only active tenant
-- managed trunks for active organizations are eligible to authenticate.
CREATE VIEW opensips_managed_trunk_credentials AS
SELECT
    tc.trunk_id,
    tc.organization_id,
    tc.username,
    tc.realm AS domain,
    tc.ha1_md5
FROM trunk_credentials AS tc
JOIN trunks AS t
  ON t.id = tc.trunk_id
 AND t.organization_id = tc.organization_id
JOIN organizations AS o
  ON o.id = tc.organization_id
WHERE t.provisioning_mode = 'managed'
  AND t.carrier_connection_id IS NULL
  AND t.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

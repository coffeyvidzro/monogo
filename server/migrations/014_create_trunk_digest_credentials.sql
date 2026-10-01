CREATE TABLE trunk_digest_credentials (
    trunk_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    direction TEXT NOT NULL,
    username TEXT NOT NULL,
    realm TEXT NOT NULL,
    ha1_md5 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (trunk_id, direction),
    CONSTRAINT fk_trunk_digest_tenant
        FOREIGN KEY (trunk_id, organization_id)
        REFERENCES trunks(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT chk_trunk_digest_direction
        CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT chk_trunk_digest_username
        CHECK (length(btrim(username)) > 0),
    CONSTRAINT chk_trunk_digest_realm
        CHECK (length(btrim(realm)) > 0),
    CONSTRAINT chk_trunk_digest_ha1
        CHECK (ha1_md5 ~ '^[0-9a-f]{32}$')
);

CREATE INDEX idx_trunk_digest_lookup
    ON trunk_digest_credentials (direction, username, realm);

CREATE UNIQUE INDEX uq_inbound_trunk_digest_identity
    ON trunk_digest_credentials (username, realm)
    WHERE direction = 'inbound';

CREATE TRIGGER set_trunk_digest_credentials_updated_at
BEFORE UPDATE ON trunk_digest_credentials
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE VIEW opensips_inbound_trunk_credentials AS
SELECT
    d.trunk_id,
    d.organization_id,
    d.username,
    d.realm AS domain,
    d.ha1_md5
FROM trunk_digest_credentials AS d
JOIN trunks AS t
  ON t.id = d.trunk_id
 AND t.organization_id = d.organization_id
JOIN organizations AS o
  ON o.id = d.organization_id
WHERE d.direction = 'inbound'
  AND t.status = 'active'
  AND t.inbound_enabled = true
  AND t.inbound_auth_method = 'digest'
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

CREATE VIEW opensips_outbound_trunk_credentials AS
SELECT
    d.trunk_id,
    d.organization_id,
    d.username,
    d.realm,
    '0x' || d.ha1_md5 AS password
FROM trunk_digest_credentials AS d
JOIN trunks AS t
  ON t.id = d.trunk_id
 AND t.organization_id = d.organization_id
JOIN organizations AS o
  ON o.id = d.organization_id
WHERE d.direction = 'outbound'
  AND t.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

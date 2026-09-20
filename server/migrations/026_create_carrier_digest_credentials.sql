-- OpenSIPS-readable digest material. This table intentionally stores only
-- realm-bound HA1 values; plaintext and decryptable carrier secrets remain in
-- the control-plane-only carrier_connections table.
CREATE TABLE carrier_digest_credentials (
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    direction TEXT NOT NULL,
    username TEXT NOT NULL,
    realm TEXT NOT NULL,
    ha1_md5 TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    PRIMARY KEY (carrier_connection_id, direction),
    CONSTRAINT chk_carrier_digest_direction CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT chk_carrier_digest_username CHECK (length(btrim(username)) > 0),
    CONSTRAINT chk_carrier_digest_realm CHECK (length(btrim(realm)) > 0),
    CONSTRAINT chk_carrier_digest_ha1 CHECK (ha1_md5 ~ '^[0-9a-f]{32}$'),
    CONSTRAINT fk_carrier_digest_tenant FOREIGN KEY (carrier_connection_id, organization_id)
        REFERENCES carrier_connections(id, organization_id) ON DELETE CASCADE
);

CREATE INDEX idx_carrier_digest_lookup
    ON carrier_digest_credentials (direction, username, realm);

-- Inbound credentials must resolve one carrier connection globally. Outbound
-- credentials are selected by connection ID and may be shared across tenants.
CREATE UNIQUE INDEX uq_inbound_carrier_digest_identity
    ON carrier_digest_credentials (username, realm)
    WHERE direction = 'inbound';

CREATE TRIGGER set_carrier_digest_credentials_updated_at
BEFORE UPDATE ON carrier_digest_credentials
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Stable read contract for the SIP data plane. uac_auth recognizes a `0x`
-- prefix as precomputed HA1, while auth_db consumes the raw hexadecimal HA1.
CREATE VIEW opensips_carrier_digest_credentials AS
SELECT
    d.carrier_connection_id,
    d.organization_id,
    d.direction,
    d.username,
    d.realm,
    CASE WHEN d.direction = 'outbound' THEN '0x' || d.ha1_md5 ELSE d.ha1_md5 END AS password
FROM carrier_digest_credentials AS d
JOIN carrier_connections AS c
  ON c.id = d.carrier_connection_id
 AND c.organization_id = d.organization_id
WHERE c.status = 'active'
  AND (
      d.direction = 'outbound'
      OR (c.inbound_enabled = true AND c.inbound_auth_method = 'digest')
  );

-- auth_db requires its configured domain and password column names, while
-- uac_auth requires a 0x-prefixed HA1. Separate views keep those contracts
-- explicit and prevent credentials from one direction being selected in the
-- other direction.
CREATE VIEW opensips_inbound_carrier_credentials AS
SELECT
    d.carrier_connection_id,
    d.organization_id,
    d.username,
    d.realm AS domain,
    d.ha1_md5
FROM carrier_digest_credentials AS d
JOIN carrier_connections AS c
  ON c.id = d.carrier_connection_id
 AND c.organization_id = d.organization_id
JOIN organizations AS o
  ON o.id = d.organization_id
WHERE d.direction = 'inbound'
  AND c.status = 'active'
  AND c.inbound_enabled = true
  AND c.inbound_auth_method = 'digest'
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

CREATE VIEW opensips_outbound_carrier_credentials AS
SELECT
    d.carrier_connection_id,
    d.organization_id,
    d.username,
    d.realm,
    '0x' || d.ha1_md5 AS password
FROM carrier_digest_credentials AS d
JOIN carrier_connections AS c
  ON c.id = d.carrier_connection_id
 AND c.organization_id = d.organization_id
JOIN organizations AS o
  ON o.id = d.organization_id
WHERE d.direction = 'outbound'
  AND c.status = 'active'
  AND o.status = 'active'
  AND o.deleted_at IS NULL;

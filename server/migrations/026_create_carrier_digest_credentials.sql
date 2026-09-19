ALTER TABLE carrier_connections
    ADD COLUMN auth_realm TEXT,
    ADD COLUMN auth_ha1_md5 TEXT,
    ADD COLUMN inbound_realm TEXT,
    ADD COLUMN inbound_ha1_md5 TEXT;

ALTER TABLE carrier_connections
    DROP CONSTRAINT chk_carrier_connections_auth_fields,
    DROP CONSTRAINT chk_carrier_connections_inbound_auth_fields;

ALTER TABLE carrier_connections
    ADD CONSTRAINT chk_carrier_connections_auth_fields CHECK (
        (outbound_auth_method = 'none' AND auth_username IS NULL AND auth_secret_ciphertext IS NULL AND auth_realm IS NULL AND auth_ha1_md5 IS NULL)
        OR
		(outbound_auth_method = 'digest' AND auth_username IS NOT NULL AND auth_secret_ciphertext IS NOT NULL AND auth_realm IS NOT NULL AND auth_ha1_md5 IS NOT NULL AND length(btrim(auth_username)) > 0 AND length(auth_secret_ciphertext) > 0 AND length(btrim(auth_realm)) > 0 AND auth_ha1_md5 ~ '^[0-9a-f]{32}$')
    ) NOT VALID,
    ADD CONSTRAINT chk_carrier_connections_inbound_auth_fields CHECK (
        (inbound_auth_method IN ('ip', 'none') AND inbound_username IS NULL AND inbound_secret_ciphertext IS NULL AND inbound_realm IS NULL AND inbound_ha1_md5 IS NULL)
        OR
		(inbound_auth_method = 'digest' AND inbound_username IS NOT NULL AND inbound_secret_ciphertext IS NOT NULL AND inbound_realm IS NOT NULL AND inbound_ha1_md5 IS NOT NULL AND length(btrim(inbound_username)) > 0 AND length(inbound_secret_ciphertext) > 0 AND length(btrim(inbound_realm)) > 0 AND inbound_ha1_md5 ~ '^[0-9a-f]{32}$')
    ) NOT VALID;

-- OpenSIPS receives one-way digest material only. The encrypted plaintext is
-- retained for controlled credential rotation but is never exposed by a view.
CREATE VIEW opensips_outbound_carrier_credentials AS
SELECT id AS carrier_connection_id,
       auth_username AS username,
       auth_realm AS realm,
       '0x' || auth_ha1_md5 AS password
FROM carrier_connections
WHERE scope = 'organization'
  AND status = 'active'
  AND outbound_auth_method = 'digest'
  AND auth_realm IS NOT NULL
  AND auth_ha1_md5 IS NOT NULL;

CREATE VIEW opensips_inbound_carrier_credentials AS
SELECT id AS carrier_connection_id,
       inbound_username AS username,
       inbound_realm AS domain,
       inbound_ha1_md5 AS ha1_md5
FROM carrier_connections
WHERE scope = 'organization'
  AND status = 'active'
  AND inbound_enabled = true
  AND inbound_auth_method = 'digest'
  AND inbound_realm IS NOT NULL
  AND inbound_ha1_md5 IS NOT NULL;

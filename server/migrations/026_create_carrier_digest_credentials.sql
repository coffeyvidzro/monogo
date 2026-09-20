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

INSERT INTO carrier_providers (slug, name, adapter, status) VALUES
('didww', 'DIDWW', 'didww', 'active')
ON CONFLICT (slug) DO NOTHING;

INSERT INTO organizations (id, name, status) VALUES
('00000000-0000-0000-0000-000000006001', 'Cloud Managed Acceptance', 'active');

INSERT INTO organization_tokens (id, organization_id, name, token_hash, token_prefix, scopes) VALUES (
    '00000000-0000-0000-0000-000000006002',
    '00000000-0000-0000-0000-000000006001',
    'cloud-managed-acceptance',
    'Y6rtC8BR465xPLxeDGcWiQyGBL6zR5L9JcqWYj8naWE',
    'lm_org_v1smoke0',
    '["numbers:read","numbers:write","voice-applications:read","voice-applications:write","calls:read","calls:write"]'::jsonb
);

INSERT INTO products (id, code, name, active) VALUES (
    '00000000-0000-0000-0000-000000006201',
    'cloud-managed-acceptance',
    'Cloud Managed Acceptance',
    true
);
INSERT INTO plans (id, product_id, code, name, active) VALUES (
    '00000000-0000-0000-0000-000000006202',
    '00000000-0000-0000-0000-000000006201',
    'managed-numbers',
    'Cloud Managed Acceptance',
    true
);
INSERT INTO prices (
    id, plan_id, pricing_type, currency, amount_minor, billing_interval,
    active, effective_from
) VALUES (
    '00000000-0000-0000-0000-000000006203',
    '00000000-0000-0000-0000-000000006202',
    'recurring', 'USD', 10000, 'month', true, now() - interval '1 day'
), (
    '00000000-0000-0000-0000-000000006204',
    '00000000-0000-0000-0000-000000006202',
    'one_time', 'USD', 2500, NULL, true, now() - interval '1 day'
);
INSERT INTO wallets (id, organization_id, currency, status) VALUES (
    '00000000-0000-0000-0000-000000006206',
    '00000000-0000-0000-0000-000000006001',
    'USD', 'active'
);
INSERT INTO wallet_ledger_entries (
    id, wallet_id, organization_id, entry_type, amount_minor,
    source_type, source_id, idempotency_key, metadata
) VALUES (
    '00000000-0000-0000-0000-000000006207',
    '00000000-0000-0000-0000-000000006206',
    '00000000-0000-0000-0000-000000006001',
    'topup', 10000,
    'acceptance_fixture', 'cloud-managed', 'cloud-managed-opening-balance', '{}'::jsonb
);

INSERT INTO organizations (id, name, status) VALUES
('00000000-0000-0000-0000-000000006101', 'Cloud Managed Isolation Tenant', 'active');
INSERT INTO organization_tokens (id, organization_id, name, token_hash, token_prefix, scopes) VALUES (
    '00000000-0000-0000-0000-000000006102',
    '00000000-0000-0000-0000-000000006101',
    'cloud-managed-isolation',
    'PsluPk0pxyRogrUeTi6j2Roc2WZi3zO_ac2M9Agd-ps',
    'lm_org_v1smoke1',
    '["numbers:read","numbers:write","voice-applications:read","voice-applications:write","calls:read","calls:write"]'::jsonb
);

INSERT INTO carrier_connections (
    id, provider_id, scope, name, status, inbound_enabled, inbound_auth_method
) VALUES (
    '00000000-0000-0000-0000-000000006010',
    (SELECT id FROM carrier_providers WHERE slug = 'didww'),
    'platform', 'Cloud managed DID ingress', 'active', true, 'ip'
);
INSERT INTO carrier_connection_provider_resources (
    carrier_connection_id, provider_id, resource_type, provider_resource_id
) VALUES (
    '00000000-0000-0000-0000-000000006010',
    (SELECT id FROM carrier_providers WHERE slug = 'didww'),
    'voice_in_trunk', 'voice-in-cloud-1'
);
INSERT INTO carrier_connection_source_ips (carrier_connection_id, cidr) VALUES
('00000000-0000-0000-0000-000000006010', '172.30.0.60/32');

INSERT INTO carrier_connections (id, provider_id, scope, name, status) VALUES (
    '00000000-0000-0000-0000-000000006020',
    (SELECT id FROM carrier_providers WHERE slug = 'generic-sip'),
    'platform', 'Managed wholesale termination', 'active'
);
INSERT INTO trunks (
    id, carrier_connection_id, provisioning_mode, name, direction, status, managed_default
) VALUES (
    '00000000-0000-0000-0000-000000006021',
    '00000000-0000-0000-0000-000000006020',
    'managed', 'Cloud managed default', 'outbound', 'active', true
);
INSERT INTO trunk_endpoints (
    id, trunk_id, host, port, transport, direction, health_status
) VALUES (
    '00000000-0000-0000-0000-000000006022',
    '00000000-0000-0000-0000-000000006021',
    '172.32.0.60', 5060, 'udp', 'outbound', 'healthy'
);

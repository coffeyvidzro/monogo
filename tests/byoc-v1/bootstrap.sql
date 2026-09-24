INSERT INTO organizations (id, name, status)
VALUES ('00000000-0000-0000-0000-000000001101', 'BYOC v1 Acceptance', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO organization_tokens (id, organization_id, name, token_hash, token_prefix, scopes)
VALUES (
    '00000000-0000-0000-0000-000000001102',
    '00000000-0000-0000-0000-000000001101',
    'byoc-v1-acceptance',
    'Y6rtC8BR465xPLxeDGcWiQyGBL6zR5L9JcqWYj8naWE',
    'lm_org_v1smoke0',
    '["calls:read","calls:write","carriers:read","carriers:write","numbers:read","numbers:write","sip-domains:read","sip-domains:write","subscribers:read","subscribers:write","trunks:read","trunks:write","voice-applications:read","voice-applications:write"]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO organizations (id, name, status)
VALUES ('00000000-0000-0000-0000-000000001201', 'BYOC v1 Acceptance Tenant B', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO organization_tokens (id, organization_id, name, token_hash, token_prefix, scopes)
VALUES (
    '00000000-0000-0000-0000-000000001202',
    '00000000-0000-0000-0000-000000001201',
    'byoc-v1-acceptance-tenant-b',
    'PsluPk0pxyRogrUeTi6j2Roc2WZi3zO_ac2M9Agd-ps',
    'lm_org_v1smoke1',
    '["calls:read","calls:write","carriers:read","carriers:write","numbers:read","numbers:write","sip-domains:read","sip-domains:write","subscribers:read","subscribers:write","trunks:read","trunks:write","voice-applications:read","voice-applications:write"]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

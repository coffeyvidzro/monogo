INSERT INTO organizations (id, name, status)
VALUES (
    '00000000-0000-0000-0000-000000001301',
    'Graceful Drain Acceptance',
    'active'
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO organization_tokens (
    id,
    organization_id,
    name,
    token_hash,
    token_prefix,
    scopes
)
VALUES (
    '00000000-0000-0000-0000-000000001302',
    '00000000-0000-0000-0000-000000001301',
    'graceful-drain-acceptance',
    'Y6rtC8BR465xPLxeDGcWiQyGBL6zR5L9JcqWYj8naWE',
    'lm_org_v1smoke0',
    '["calls:read","calls:write","numbers:read","numbers:write","trunks:read","trunks:write"]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

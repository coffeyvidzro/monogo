INSERT INTO organizations (id, name, status)
VALUES ('00000000-0000-0000-0000-000000001101', 'Voice Agent v1 Acceptance', 'active')
ON CONFLICT (id) DO NOTHING;

INSERT INTO organization_tokens (
    id, organization_id, name, token_hash, token_prefix, scopes
)
VALUES (
    '00000000-0000-0000-0000-000000001102',
    '00000000-0000-0000-0000-000000001101',
    'voice-agent-v1-acceptance',
    'Y6rtC8BR465xPLxeDGcWiQyGBL6zR5L9JcqWYj8naWE',
    'lm_org_v1smoke0',
    '["calls:read","calls:write","carriers:read","carriers:write","numbers:read","numbers:write","trunks:read","trunks:write","voice-applications:read","voice-applications:write","voice-agents:read","voice-agents:write"]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO organizations (id, name, status)
VALUES ('00000000-0000-0000-0000-000000001001', 'Voice v1 Acceptance', 'active')
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
    '00000000-0000-0000-0000-000000001002',
    '00000000-0000-0000-0000-000000001001',
    'voice-v1-acceptance',
    'Y6rtC8BR465xPLxeDGcWiQyGBL6zR5L9JcqWYj8naWE',
    'lm_org_v1smoke0',
    '["audit:read","calls:read","calls:write","carriers:read","carriers:write","conferences:read","conferences:write","numbers:read","numbers:write","recordings:read","recordings:write","sip-domains:read","sip-domains:write","subscribers:read","subscribers:write","trunks:read","trunks:write","voice-applications:read","voice-applications:write","webhooks:read","webhooks:write"]'::jsonb
)
ON CONFLICT (id) DO NOTHING;

INSERT INTO subscription_plans (
    id,
    code,
    name,
    currency,
    interval,
    amount_micros,
    status
)
VALUES (
    '00000000-0000-0000-0000-000000009001',
    'acceptance',
    'Acceptance',
    'USD',
    'month',
    0,
    'active'
)
ON CONFLICT (id) DO NOTHING;


INSERT INTO subscriptions (
    id,
    organization_id,
    plan_id,
    status,
    currency,
    amount_micros,
    interval,
    current_period_start,
    current_period_end,
    started_at
)
VALUES (
    '00000000-0000-0000-0000-000000001009',
    '00000000-0000-0000-0000-000000001001',
    '00000000-0000-0000-0000-000000009001',
    'active',
    'USD',
    0,
    'month',
    now() - INTERVAL '1 day',
    now() + INTERVAL '1 year',
    now() - INTERVAL '1 day'
)
ON CONFLICT (id) DO NOTHING;

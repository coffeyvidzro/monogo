-- Leamout Managed Carrier keeps number ownership and voice transport as
-- separate provider responsibilities. These records identify the adapters;
-- account-specific connections, trunks, endpoints, and secrets are provisioned
-- independently by authorized platform operations.
INSERT INTO carrier_providers (id, slug, name, adapter, status)
VALUES
    (
        'd1d00000-0000-4000-8000-000000000001',
        'didww',
        'DIDWW',
        'didww',
        'active'
    ),
    (
        'c011ea00-0000-4000-8000-000000000001',
        'commpeak',
        'CommPeak',
        'commpeak',
        'active'
    )
ON CONFLICT (slug) DO UPDATE
SET
    name = EXCLUDED.name,
    adapter = EXCLUDED.adapter,
    status = EXCLUDED.status;

-- DIDWW is the initial managed DID acquisition and inbound provider.
INSERT INTO carrier_providers (slug, name, adapter, status)
VALUES ('didww', 'DIDWW', 'didww', 'active')
ON CONFLICT (slug) DO NOTHING;

-- A purchase remains separate from the resulting phone number so an uncertain
-- provider response can always be reconciled without issuing a second order.
CREATE TABLE managed_number_orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    phone_number_id UUID,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    number TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    available_did_id TEXT NOT NULL,
    sku_id TEXT NOT NULL,
    provider_order_id TEXT,
    provider_did_id TEXT,
    inbound_trunk_id TEXT,
    status TEXT NOT NULL DEFAULT 'ready',
    submitted_at TIMESTAMPTZ,
    ownership_verified_at TIMESTAMPTZ,
    routing_verified_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    reconcile_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    reconcile_attempts INTEGER NOT NULL DEFAULT 0,
    error_code TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    UNIQUE (organization_id, idempotency_key),
    FOREIGN KEY (phone_number_id, organization_id, provider_id)
        REFERENCES phone_numbers(id, organization_id, provider_id),
    CHECK (length(idempotency_key) BETWEEN 1 AND 255 AND idempotency_key = btrim(idempotency_key)),
    CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CHECK (number ~ '^\\+[1-9][0-9]{6,14}$'),
    CHECK (country_code ~ '^[A-Z]{2}$'),
    CHECK (length(btrim(available_did_id)) > 0 AND length(btrim(sku_id)) > 0),
    CHECK (provider_order_id IS NULL OR length(btrim(provider_order_id)) > 0),
    CHECK (provider_did_id IS NULL OR length(btrim(provider_did_id)) > 0),
    CHECK (inbound_trunk_id IS NULL OR length(btrim(inbound_trunk_id)) > 0),
    CHECK (reconcile_attempts >= 0),
    CHECK (status IN ('ready', 'submitting', 'outcome_unknown', 'provider_pending',
                      'configuring', 'completed', 'failed', 'manual_review')),
    CHECK (status NOT IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring', 'completed')
           OR submitted_at IS NOT NULL),
    CHECK (status NOT IN ('provider_pending', 'configuring', 'completed') OR provider_order_id IS NOT NULL),
    CHECK (status NOT IN ('configuring', 'completed')
           OR (provider_did_id IS NOT NULL AND ownership_verified_at IS NOT NULL)),
    CHECK (status <> 'completed' OR (phone_number_id IS NOT NULL AND inbound_trunk_id IS NOT NULL
           AND routing_verified_at IS NOT NULL AND activated_at IS NOT NULL))
);

-- Failed pre-submission attempts release their claim. Every ambiguous or
-- successful attempt continues to block duplicate acquisition globally.
CREATE UNIQUE INDEX uq_managed_number_orders_live_number
    ON managed_number_orders (number) WHERE status <> 'failed';
CREATE UNIQUE INDEX uq_managed_number_orders_live_inventory
    ON managed_number_orders (provider_id, available_did_id) WHERE status <> 'failed';
CREATE UNIQUE INDEX uq_managed_number_orders_provider_order
    ON managed_number_orders (provider_id, provider_order_id) WHERE provider_order_id IS NOT NULL;
CREATE UNIQUE INDEX uq_managed_number_orders_provider_did
    ON managed_number_orders (provider_id, provider_did_id) WHERE provider_did_id IS NOT NULL;
CREATE INDEX idx_managed_number_orders_org_created
    ON managed_number_orders (organization_id, created_at DESC);
CREATE INDEX idx_managed_number_orders_reconciliation
    ON managed_number_orders (reconcile_after, updated_at)
    WHERE status IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring');

CREATE TRIGGER set_managed_number_orders_updated_at
BEFORE UPDATE ON managed_number_orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

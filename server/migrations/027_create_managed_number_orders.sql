-- The provider record identifies DIDWW as Leamout's initial managed DID
-- acquisition and inbound provider, separate from CommPeak termination.
INSERT INTO carrier_providers (slug, name, adapter, status)
VALUES ('didww', 'DIDWW', 'didww', 'active')
ON CONFLICT (slug) DO NOTHING;

-- Acquisition attempts must survive retries, worker crashes, and uncertain
-- provider outcomes. A purchase is not the same entity as an owned phone number.
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
    quote_id UUID NOT NULL,
    purchase_amount_minor BIGINT NOT NULL,
    recurring_amount_minor BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    quote_expires_at TIMESTAMPTZ NOT NULL,
    wallet_reservation_id UUID,
    provider_order_id TEXT,
    provider_did_id TEXT,
    status TEXT NOT NULL DEFAULT 'pending_funding',
    submitted_at TIMESTAMPTZ,
    error_code TEXT,
    error_message TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_managed_number_orders_idempotency
        UNIQUE (organization_id, idempotency_key),
    CONSTRAINT uq_managed_number_orders_reservation UNIQUE (wallet_reservation_id),
    CONSTRAINT fk_managed_number_orders_phone_number
        FOREIGN KEY (phone_number_id, organization_id, provider_id)
        REFERENCES phone_numbers(id, organization_id, provider_id),
    CONSTRAINT chk_managed_number_orders_key
        CHECK (length(idempotency_key) BETWEEN 1 AND 255 AND idempotency_key = btrim(idempotency_key)),
    CONSTRAINT chk_managed_number_orders_hash
        CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_managed_number_orders_number
        CHECK (number ~ '^\\+[1-9][0-9]{6,14}$'),
    CONSTRAINT chk_managed_number_orders_country
        CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_managed_number_orders_inventory
        CHECK (length(btrim(available_did_id)) > 0 AND length(btrim(sku_id)) > 0),
    CONSTRAINT chk_managed_number_orders_price
        CHECK (purchase_amount_minor > 0 AND recurring_amount_minor >= 0 AND currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_managed_number_orders_quote
        CHECK (quote_expires_at > created_at),
    CONSTRAINT chk_managed_number_orders_provider_order
        CHECK (provider_order_id IS NULL OR length(btrim(provider_order_id)) > 0),
    CONSTRAINT chk_managed_number_orders_provider_did
        CHECK (provider_did_id IS NULL OR length(btrim(provider_did_id)) > 0),
    CONSTRAINT chk_managed_number_orders_status
        CHECK (status IN (
            'pending_funding', 'ready', 'submitting', 'outcome_unknown',
            'provider_pending', 'configuring', 'completed', 'failed', 'manual_review'
        )),
    CONSTRAINT chk_managed_number_orders_funding
        CHECK (status <> 'ready' OR wallet_reservation_id IS NOT NULL),
    CONSTRAINT chk_managed_number_orders_submission
        CHECK (status NOT IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring', 'completed')
            OR (wallet_reservation_id IS NOT NULL AND submitted_at IS NOT NULL)),
    CONSTRAINT chk_managed_number_orders_provider_state
        CHECK (status NOT IN ('provider_pending', 'configuring', 'completed')
            OR provider_order_id IS NOT NULL),
    CONSTRAINT chk_managed_number_orders_did_state
        CHECK (status NOT IN ('configuring', 'completed') OR provider_did_id IS NOT NULL),
    CONSTRAINT chk_managed_number_orders_completion
        CHECK (status <> 'completed' OR phone_number_id IS NOT NULL)
);

-- A DID can have only one unresolved or assigned purchase across tenants,
-- even if different inventory IDs refer to the same E.164 number. Confirmed
-- failures release their claim; ambiguous outcomes do not.
CREATE UNIQUE INDEX uq_managed_number_orders_live_number
    ON managed_number_orders (number)
    WHERE status <> 'failed';

CREATE UNIQUE INDEX uq_managed_number_orders_live_inventory
    ON managed_number_orders (provider_id, available_did_id)
    WHERE status <> 'failed';

CREATE UNIQUE INDEX uq_managed_number_orders_provider_order
    ON managed_number_orders (provider_id, provider_order_id)
    WHERE provider_order_id IS NOT NULL;

CREATE UNIQUE INDEX uq_managed_number_orders_provider_did
    ON managed_number_orders (provider_id, provider_did_id)
    WHERE provider_did_id IS NOT NULL;

CREATE INDEX idx_managed_number_orders_org_created
    ON managed_number_orders (organization_id, created_at DESC);

CREATE INDEX idx_managed_number_orders_reconciliation
    ON managed_number_orders (status, updated_at)
    WHERE status IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring', 'manual_review');

CREATE TRIGGER set_managed_number_orders_updated_at
BEFORE UPDATE ON managed_number_orders
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE managed_number_orders IS
    'Durable internal DIDWW acquisition intents; does not itself authorize provider orders or wallet charges.';

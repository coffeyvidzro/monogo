-- A checkout can have multiple provider attempts; each upstream attempt
-- needs a durable internal identity before initiating an external charge.
CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_id UUID NOT NULL REFERENCES checkouts(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    attempt_key TEXT NOT NULL,
    provider_reference TEXT,
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'created',
    verified_at TIMESTAMPTZ,
    wallet_transaction_id UUID UNIQUE REFERENCES wallet_transactions(id) ON DELETE RESTRICT,
    failure_code TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_payments_attempt UNIQUE (checkout_id, provider, attempt_key),
    CONSTRAINT chk_payments_provider CHECK (provider IN ('stripe', 'paystack')),
    CONSTRAINT chk_payments_attempt_key CHECK (
        length(attempt_key) BETWEEN 1 AND 255 AND attempt_key = btrim(attempt_key)
    ),
    CONSTRAINT chk_payments_provider_ref CHECK (
        provider_reference IS NULL OR length(btrim(provider_reference)) > 0
    ),
    CONSTRAINT chk_payments_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_payments_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_payments_status CHECK (
        status IN ('created', 'pending', 'succeeded', 'failed', 'canceled')
    ),
    CONSTRAINT chk_payments_verified CHECK (
        status <> 'succeeded' OR verified_at IS NOT NULL
    ),
    CONSTRAINT chk_payments_credit CHECK (
        wallet_transaction_id IS NULL OR (status = 'succeeded' AND verified_at IS NOT NULL)
    )
);

CREATE UNIQUE INDEX uq_payments_provider_reference
    ON payments (provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

-- A checkout must never be credited by two successful payment attempts.
-- If a second external payment succeeds, reconcile/refund it separately.
CREATE UNIQUE INDEX uq_payments_credited_checkout
    ON payments (checkout_id)
    WHERE wallet_transaction_id IS NOT NULL;

CREATE INDEX idx_payments_checkout_created
    ON payments (checkout_id, created_at DESC);
CREATE INDEX idx_payments_uncredited_succeeded
    ON payments (updated_at)
    WHERE status = 'succeeded' AND wallet_transaction_id IS NULL;

CREATE TRIGGER set_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Inbound provider notifications, not the tenant-facing webhook_events table.
-- Paystack does not provide a guaranteed unique webhook event ID for every
-- delivery: use a deterministic provider reference + event-type key where
-- appropriate, and ALWAYS independently verify financial settlement.
CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID REFERENCES payments(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload_sha256 TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'received',
    error_code TEXT,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT uq_payment_events_provider_identity UNIQUE (provider, provider_event_id),
    CONSTRAINT chk_payment_events_provider CHECK (provider IN ('stripe', 'paystack')),
    CONSTRAINT chk_payment_events_event_id CHECK (length(btrim(provider_event_id)) > 0),
    CONSTRAINT chk_payment_events_type CHECK (length(btrim(event_type)) > 0),
    CONSTRAINT chk_payment_events_digest CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_payment_events_status CHECK (status IN ('received', 'processed', 'ignored', 'failed')),
    CONSTRAINT chk_payment_events_processing CHECK (
        (status = 'received' AND processed_at IS NULL)
        OR (status <> 'received' AND processed_at IS NOT NULL)
    )
);

CREATE INDEX idx_payment_events_payment_received
    ON payment_events (payment_id, received_at DESC);
CREATE INDEX idx_payment_events_unprocessed
    ON payment_events (received_at) WHERE status = 'received';

COMMENT ON TABLE payments IS
    'Provider payment attempts. A succeeded payment is not a wallet credit until a linked immutable ledger transaction exists.';
COMMENT ON TABLE payment_events IS
    'Signature-verified inbound provider event metadata and raw payload digest. Never store payment credentials or unredacted provider payloads here.';

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    checkout_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    provider TEXT NOT NULL,
    attempt INTEGER NOT NULL,
    provider_payment_id TEXT,
    amount_micros BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    failure_code TEXT,
    paid_at TIMESTAMPTZ,
    metadata JSONB NOT NULL DEFAULT '{}'::jsonb,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_payments_checkout
        FOREIGN KEY (checkout_id, organization_id)
        REFERENCES checkouts (id, organization_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_payments_id_organization
        UNIQUE (id, organization_id),
    CONSTRAINT uq_payments_checkout_attempt
        UNIQUE (checkout_id, attempt),
    CONSTRAINT chk_payments_provider
        CHECK (provider IN ('stripe', 'paystack')),
    CONSTRAINT chk_payments_attempt
        CHECK (attempt > 0),
    CONSTRAINT chk_payments_provider_payment_id
        CHECK (
            provider_payment_id IS NULL
            OR length(btrim(provider_payment_id)) > 0
        ),
    CONSTRAINT chk_payments_amount
        CHECK (amount_micros > 0),
    CONSTRAINT chk_payments_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_payments_status
        CHECK (
            status IN (
                'pending',
                'processing',
                'succeeded',
                'failed',
                'cancelled',
                'refunded',
                'partially_refunded'
            )
        ),
    CONSTRAINT chk_payments_paid_at
        CHECK (
            status NOT IN ('succeeded', 'refunded', 'partially_refunded')
            OR paid_at IS NOT NULL
        ),
    CONSTRAINT chk_payments_metadata
        CHECK (jsonb_typeof(metadata) = 'object')
);

CREATE UNIQUE INDEX uq_payments_provider_payment_id
    ON payments (provider, provider_payment_id)
    WHERE provider_payment_id IS NOT NULL;

CREATE UNIQUE INDEX uq_payments_active_checkout
    ON payments (checkout_id)
    WHERE status IN ('pending', 'processing');

CREATE INDEX idx_payments_checkout_created
    ON payments (checkout_id, created_at DESC);

CREATE INDEX idx_payments_organization_created
    ON payments (organization_id, created_at DESC);

CREATE TRIGGER set_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

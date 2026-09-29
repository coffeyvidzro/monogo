-- Provider-neutral payment orchestration.
-- Payments collect money for exactly one Monogo obligation: a platform
-- subscription period or a prepaid wallet top-up.

CREATE TABLE payments (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    purpose TEXT NOT NULL,
    provider TEXT NOT NULL,
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    amount_micros BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_reference TEXT,
    provider_event_id TEXT,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    failure_code TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_payments_purpose
        CHECK (purpose IN ('subscription', 'wallet_topup')),
    CONSTRAINT chk_payments_provider
        CHECK (provider ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_payments_amount
        CHECK (amount_micros > 0),
    CONSTRAINT chk_payments_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_payments_status
        CHECK (status IN ('pending', 'succeeded', 'failed', 'cancelled')),
    CONSTRAINT chk_payments_subscription_shape
        CHECK (
            (
                purpose = 'subscription'
                AND subscription_id IS NOT NULL
                AND period_start IS NOT NULL
                AND period_end IS NOT NULL
                AND period_end > period_start
            )
            OR
            (
                purpose = 'wallet_topup'
                AND subscription_id IS NULL
                AND period_start IS NULL
                AND period_end IS NULL
            )
        ),
    CONSTRAINT chk_payments_completion
        CHECK (
            (status = 'pending' AND completed_at IS NULL)
            OR
            (status <> 'pending' AND completed_at IS NOT NULL)
        ),
    CONSTRAINT chk_payments_failure
        CHECK (
            (status = 'failed' AND failure_code IS NOT NULL)
            OR
            (status <> 'failed' AND failure_code IS NULL)
        )
);

CREATE UNIQUE INDEX uq_payments_provider_reference
    ON payments (provider, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE UNIQUE INDEX uq_payments_provider_event
    ON payments (provider, provider_event_id)
    WHERE provider_event_id IS NOT NULL;

CREATE UNIQUE INDEX uq_payments_pending_subscription
    ON payments (subscription_id)
    WHERE purpose = 'subscription' AND status = 'pending';

CREATE INDEX idx_payments_organization_created
    ON payments (organization_id, created_at DESC);

CREATE TRIGGER set_payments_updated_at
BEFORE UPDATE ON payments
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE checkouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    purpose TEXT NOT NULL,
    subscription_id UUID NOT NULL REFERENCES subscriptions(id) ON DELETE RESTRICT,
    reference TEXT NOT NULL UNIQUE,
    amount_micros BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider TEXT,
    payment_method TEXT,
    next_action TEXT NOT NULL DEFAULT 'wait',
    provider_message TEXT,
    expires_at TIMESTAMPTZ NOT NULL,
    failure_code TEXT,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_checkouts_id_organization
        UNIQUE (id, organization_id),
    CONSTRAINT chk_checkouts_purpose
        CHECK (purpose = 'subscription'),
    CONSTRAINT chk_checkouts_reference
        CHECK (reference ~ '^[A-Za-z0-9._=-]+$'),
    CONSTRAINT chk_checkouts_amount
        CHECK (amount_micros > 0),
    CONSTRAINT chk_checkouts_currency
        CHECK (currency = 'USD'),
    CONSTRAINT chk_checkouts_status
        CHECK (status IN ('pending', 'processing', 'succeeded', 'failed', 'cancelled', 'expired')),
    CONSTRAINT chk_checkouts_payment_binding
        CHECK (
            (provider IS NULL AND payment_method IS NULL)
            OR (provider = 'stripe' AND payment_method = 'card')
        ),
    CONSTRAINT chk_checkouts_action
        CHECK (next_action IN ('none', 'wait')),
    CONSTRAINT chk_checkouts_message
        CHECK (
            provider_message IS NULL
            OR length(btrim(provider_message)) > 0
        ),
    CONSTRAINT chk_checkouts_expiry
        CHECK (expires_at > created_at),
    CONSTRAINT chk_checkouts_completion
        CHECK (
            (
                status IN ('pending', 'processing')
                AND completed_at IS NULL
            )
            OR (
                status IN ('succeeded', 'failed', 'cancelled', 'expired')
                AND completed_at IS NOT NULL
            )
        ),
    CONSTRAINT chk_checkouts_terminal_action
        CHECK (
            status IN ('pending', 'processing')
            OR next_action = 'none'
        ),
    CONSTRAINT chk_checkouts_failure
        CHECK (
            (status = 'failed' AND failure_code IS NOT NULL)
            OR (status <> 'failed' AND failure_code IS NULL)
        )
);

CREATE UNIQUE INDEX uq_checkouts_active_subscription
    ON checkouts (subscription_id)
    WHERE purpose = 'subscription'
      AND status IN ('pending', 'processing');

CREATE INDEX idx_checkouts_organization_created
    ON checkouts (organization_id, created_at DESC);

CREATE INDEX idx_checkouts_pending_expiry
    ON checkouts (expires_at, created_at)
    WHERE status IN ('pending', 'processing');

CREATE TRIGGER set_checkouts_updated_at
BEFORE UPDATE ON checkouts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

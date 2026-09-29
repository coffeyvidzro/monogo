CREATE TABLE checkouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    purpose TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'open',
    subscription_id UUID REFERENCES subscriptions(id) ON DELETE RESTRICT,
    amount_micros BIGINT NOT NULL,
    currency CHAR(3) NOT NULL,
    period_start TIMESTAMPTZ,
    period_end TIMESTAMPTZ,
    failure_code TEXT,
    confirmed_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    failed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_checkouts_id_organization UNIQUE (id, organization_id),
    CONSTRAINT chk_checkouts_purpose CHECK (purpose IN ('subscription', 'wallet_topup')),
    CONSTRAINT chk_checkouts_status CHECK (status IN ('open', 'processing', 'completed', 'failed', 'cancelled')),
    CONSTRAINT chk_checkouts_amount CHECK (amount_micros > 0),
    CONSTRAINT chk_checkouts_currency CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE UNIQUE INDEX uq_checkouts_active_subscription
    ON checkouts (subscription_id)
    WHERE purpose = 'subscription'
      AND status IN ('open', 'processing');

CREATE INDEX idx_checkouts_organization_created
    ON checkouts (organization_id, created_at DESC);

CREATE TRIGGER set_checkouts_updated_at
BEFORE UPDATE ON checkouts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

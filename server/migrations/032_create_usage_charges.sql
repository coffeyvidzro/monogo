-- A usage charge records the trusted, rated monetary result of one immutable
-- usage observation. The linked wallet transaction is the actual debit.
CREATE TABLE usage_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    usage_event_id UUID NOT NULL UNIQUE REFERENCES usage_events(id) ON DELETE RESTRICT,
    wallet_transaction_id UUID NOT NULL UNIQUE REFERENCES wallet_transactions(id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_usage_charges_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_usage_charges_currency CHECK (currency ~ '^[A-Z]{3}$')
);

CREATE INDEX idx_usage_charges_created ON usage_charges (created_at DESC);

COMMENT ON TABLE usage_charges IS
    'A trusted rated usage amount atomically debited from a prepaid wallet; one charge per immutable usage event.';

-- A checkout is the customer's intent to add money, not evidence that any
-- payment has succeeded. The wallet records the organization and currency.
CREATE TABLE checkouts (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    amount_minor BIGINT NOT NULL,
    currency TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    credited_transaction_id UUID UNIQUE REFERENCES wallet_transactions(id) ON DELETE RESTRICT,
    expires_at TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_checkouts_wallet_idempotency UNIQUE (wallet_id, idempotency_key),
    CONSTRAINT chk_checkouts_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_checkouts_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_checkouts_key CHECK (
        length(idempotency_key) BETWEEN 1 AND 255
        AND idempotency_key = btrim(idempotency_key)
    ),
    CONSTRAINT chk_checkouts_hash CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_checkouts_expiration CHECK (expires_at > created_at),
    CONSTRAINT chk_checkouts_status CHECK (status IN ('pending', 'completed', 'expired', 'canceled')),
    CONSTRAINT chk_checkouts_completion CHECK (
        (status = 'completed' AND credited_transaction_id IS NOT NULL AND completed_at IS NOT NULL)
        OR (status <> 'completed' AND credited_transaction_id IS NULL AND completed_at IS NULL)
    )
);

CREATE INDEX idx_checkouts_wallet_created ON checkouts (wallet_id, created_at DESC);
CREATE INDEX idx_checkouts_pending_expiration
    ON checkouts (expires_at) WHERE status = 'pending';

CREATE TRIGGER set_checkouts_updated_at
BEFORE UPDATE ON checkouts
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE checkouts IS
    'Wallet top-up intentions. Completion requires a verified payment and a unique wallet ledger credit; never infer success from checkout creation.';

-- Prepaid PAYG balance, scoped to one organization and currency.
-- Financial records are retained even if an organization is disabled.
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    currency TEXT NOT NULL,
    balance_minor BIGINT NOT NULL DEFAULT 0,
    reserved_minor BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallets_organization_currency UNIQUE (organization_id, currency),
    CONSTRAINT uq_wallets_id_organization UNIQUE (id, organization_id),
    CONSTRAINT chk_wallets_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_wallets_nonnegative_balance CHECK (balance_minor >= 0),
    CONSTRAINT chk_wallets_reserved_nonnegative CHECK (reserved_minor >= 0),
    CONSTRAINT chk_wallets_reserved_within_balance CHECK (reserved_minor <= balance_minor)
);

CREATE INDEX idx_wallets_organization ON wallets (organization_id);

CREATE TRIGGER set_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- One immutable entry per successful balance change. Pending checkout attempts
-- live in the payment domain; a verified top-up becomes a credit here.
CREATE TABLE wallet_transactions (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL REFERENCES wallets(id) ON DELETE RESTRICT,
    direction TEXT NOT NULL,
    reason TEXT NOT NULL,
    amount_minor BIGINT NOT NULL,
    balance_after_minor BIGINT NOT NULL,
    reference_type TEXT NOT NULL,
    reference_id UUID NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallet_transactions_reference
        UNIQUE (wallet_id, reference_type, reference_id),
    CONSTRAINT chk_wallet_transactions_direction
        CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT chk_wallet_transactions_reason
        CHECK (reason ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_wallet_transactions_amount
        CHECK (amount_minor > 0),
    CONSTRAINT chk_wallet_transactions_balance
        CHECK (balance_after_minor >= 0),
    CONSTRAINT chk_wallet_transactions_reference_type
        CHECK (reference_type ~ '^[a-z][a-z0-9_]{0,63}$')
);

CREATE INDEX idx_wallet_transactions_wallet_created
    ON wallet_transactions (wallet_id, created_at DESC, id DESC);

-- Reversals/refunds must create new credit entries, not rewrite history.
CREATE FUNCTION reject_wallet_transaction_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wallet transactions are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER wallet_transactions_immutable
BEFORE UPDATE OR DELETE ON wallet_transactions
FOR EACH ROW EXECUTE FUNCTION reject_wallet_transaction_mutation();

-- Reservations make funds unavailable before a managed-provider obligation or
-- long-running communication is allowed to consume them. Capture is final and
-- releases any unused part of the hold.
CREATE TABLE wallet_reservations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    amount_minor BIGINT NOT NULL,
    captured_amount_minor BIGINT,
    operation_type TEXT NOT NULL,
    operation_id TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    expires_at TIMESTAMPTZ NOT NULL,
    captured_transaction_id UUID UNIQUE REFERENCES wallet_transactions(id) ON DELETE RESTRICT,
    captured_at TIMESTAMPTZ,
    released_at TIMESTAMPTZ,
    expired_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_reservations_wallet_organization
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT uq_wallet_reservations_operation
        UNIQUE (organization_id, operation_type, operation_id),
    CONSTRAINT chk_wallet_reservations_amount CHECK (amount_minor > 0),
    CONSTRAINT chk_wallet_reservations_captured_amount CHECK (
        captured_amount_minor IS NULL OR captured_amount_minor BETWEEN 1 AND amount_minor
    ),
    CONSTRAINT chk_wallet_reservations_operation_type CHECK (
        operation_type ~ '^[a-z0-9]+(?:_[a-z0-9]+)*$'
    ),
    CONSTRAINT chk_wallet_reservations_operation_id CHECK (
        length(operation_id) BETWEEN 1 AND 255 AND operation_id = btrim(operation_id)
    ),
    CONSTRAINT chk_wallet_reservations_status CHECK (
        status IN ('active', 'captured', 'released', 'expired')
    ),
    CONSTRAINT chk_wallet_reservations_expiry CHECK (expires_at > created_at),
    CONSTRAINT chk_wallet_reservations_lifecycle CHECK (
        (status = 'active' AND captured_amount_minor IS NULL
            AND captured_transaction_id IS NULL AND captured_at IS NULL
            AND released_at IS NULL AND expired_at IS NULL)
        OR (status = 'captured' AND captured_amount_minor IS NOT NULL
            AND captured_transaction_id IS NOT NULL AND captured_at IS NOT NULL
            AND released_at IS NULL AND expired_at IS NULL)
        OR (status = 'released' AND captured_amount_minor IS NULL
            AND captured_transaction_id IS NULL AND captured_at IS NULL
            AND released_at IS NOT NULL AND expired_at IS NULL)
        OR (status = 'expired' AND captured_amount_minor IS NULL
            AND captured_transaction_id IS NULL AND captured_at IS NULL
            AND released_at IS NULL AND expired_at IS NOT NULL)
    )
);

CREATE INDEX idx_wallet_reservations_active_expiry
    ON wallet_reservations (expires_at, id) WHERE status = 'active';
CREATE INDEX idx_wallet_reservations_wallet_created
    ON wallet_reservations (wallet_id, created_at DESC, id DESC);

CREATE TRIGGER set_wallet_reservations_updated_at
BEFORE UPDATE ON wallet_reservations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

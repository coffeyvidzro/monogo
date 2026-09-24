-- Prepaid PAYG balance, scoped to one organization and currency.
-- Financial records are retained even if an organization is disabled.
CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    currency TEXT NOT NULL,
    balance_minor BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallets_organization_currency UNIQUE (organization_id, currency),
    CONSTRAINT chk_wallets_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_wallets_nonnegative_balance CHECK (balance_minor >= 0)
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

COMMENT ON TABLE wallets IS
    'Available prepaid PAYG funds for an organization in a single currency; updates must be posted with a ledger entry in the same transaction.';

COMMENT ON TABLE wallet_transactions IS
    'Immutable successful prepaid credits and debits. A unique business reference prevents a repeated operation from changing balance twice.';

-- One prepaid USD wallet per organization.
-- Money uses integer micros: 1 USD = 1,000,000 micros.

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    status TEXT NOT NULL DEFAULT 'active',
    balance_micros BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallets_organization
        UNIQUE (organization_id),
    CONSTRAINT uq_wallets_id_organization
        UNIQUE (id, organization_id),
    CONSTRAINT chk_wallets_currency
        CHECK (currency = 'USD'),
    CONSTRAINT chk_wallets_status
        CHECK (status IN ('active', 'frozen', 'closed')),
    CONSTRAINT chk_wallets_balance_nonnegative
        CHECK (balance_micros >= 0)
);

CREATE INDEX idx_wallets_organization
    ON wallets (organization_id, created_at DESC);

CREATE TRIGGER set_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Immutable settled movements of prepaid customer funds.
-- A reference identifies the business resource that caused the movement
-- without introducing a separate generic charge entity.

CREATE TABLE wallet_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    operation_id UUID NOT NULL,
    direction TEXT NOT NULL,
    reason TEXT NOT NULL,
    amount_micros BIGINT NOT NULL,
    balance_after_micros BIGINT NOT NULL,
    reference_type TEXT,
    reference_id UUID,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_ledger_wallet
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT uq_wallet_ledger_operation
        UNIQUE (operation_id),
    CONSTRAINT chk_wallet_ledger_direction
        CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT chk_wallet_ledger_reason
        CHECK (reason ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_wallet_ledger_amount
        CHECK (amount_micros > 0),
    CONSTRAINT chk_wallet_ledger_balance
        CHECK (balance_after_micros >= 0),
    CONSTRAINT chk_wallet_ledger_reference
        CHECK (
            (reference_type IS NULL AND reference_id IS NULL)
            OR
            (
                reference_type ~ '^[a-z][a-z0-9_]{0,63}$'
                AND reference_id IS NOT NULL
            )
        )
);

CREATE INDEX idx_wallet_ledger_wallet_created
    ON wallet_ledger_entries (wallet_id, occurred_at DESC, id DESC);

CREATE INDEX idx_wallet_ledger_reference
    ON wallet_ledger_entries (reference_type, reference_id)
    WHERE reference_id IS NOT NULL;

CREATE FUNCTION reject_wallet_ledger_entry_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wallet ledger entries are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER wallet_ledger_entries_immutable
BEFORE UPDATE OR DELETE ON wallet_ledger_entries
FOR EACH ROW EXECUTE FUNCTION reject_wallet_ledger_entry_mutation();

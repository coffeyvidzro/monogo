-- Settled balance movements form the immutable financial ledger.
-- Reservations do not create ledger entries because they do not move settled
-- money. Refunds and corrections create new entries instead of rewriting old
-- history.

CREATE TABLE wallet_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_event_id UUID NOT NULL,
    wallet_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    charge_id UUID,
    direction TEXT NOT NULL,
    reason TEXT NOT NULL,
    amount_micros BIGINT NOT NULL,
    balance_after_micros BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_ledger_wallet
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_charge
        FOREIGN KEY (charge_id, organization_id, wallet_id)
        REFERENCES charges (id, organization_id, wallet_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_event
        FOREIGN KEY (wallet_event_id, organization_id, wallet_id)
        REFERENCES wallet_events (id, organization_id, wallet_id) ON DELETE RESTRICT,
    CONSTRAINT uq_wallet_ledger_event
        UNIQUE (wallet_event_id),
    CONSTRAINT chk_wallet_ledger_direction
        CHECK (direction IN ('credit', 'debit')),
    CONSTRAINT chk_wallet_ledger_reason
        CHECK (reason ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_wallet_ledger_amount
        CHECK (amount_micros > 0),
    CONSTRAINT chk_wallet_ledger_balance
        CHECK (balance_after_micros >= 0)
);

CREATE INDEX idx_wallet_ledger_wallet_created
    ON wallet_ledger_entries (wallet_id, occurred_at DESC, id DESC);

CREATE INDEX idx_wallet_ledger_charge_created
    ON wallet_ledger_entries (charge_id, occurred_at DESC, id DESC)
    WHERE charge_id IS NOT NULL;

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

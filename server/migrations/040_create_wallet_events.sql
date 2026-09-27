-- Every successful atomic OCS mutation is mirrored as one immutable event.
-- wallet_version orders all wallet mutations; charge_sequence independently
-- orders mutations belonging to one long-running charge.

CREATE TABLE wallet_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    charge_id UUID,
    operation_id UUID NOT NULL,
    wallet_version BIGINT NOT NULL,
    charge_sequence BIGINT,
    event_type TEXT NOT NULL,
    balance_delta_micros BIGINT NOT NULL DEFAULT 0,
    reserved_delta_micros BIGINT NOT NULL DEFAULT 0,
    balance_after_micros BIGINT NOT NULL,
    reserved_after_micros BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_events_wallet
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_events_charge
        FOREIGN KEY (charge_id, organization_id, wallet_id)
        REFERENCES charges (id, organization_id, wallet_id) ON DELETE RESTRICT,
    CONSTRAINT uq_wallet_events_id_organization_wallet
        UNIQUE (id, organization_id, wallet_id),
    CONSTRAINT uq_wallet_events_operation
        UNIQUE (operation_id),
    CONSTRAINT uq_wallet_events_wallet_version
        UNIQUE (wallet_id, wallet_version),
    CONSTRAINT chk_wallet_events_wallet_version
        CHECK (wallet_version > 0),
    CONSTRAINT chk_wallet_events_charge_sequence
        CHECK (
            (charge_id IS NULL AND charge_sequence IS NULL)
            OR
            (charge_id IS NOT NULL AND charge_sequence IS NOT NULL AND charge_sequence > 0)
        ),
    CONSTRAINT chk_wallet_events_type
        CHECK (
            event_type IN (
                'credit',
                'reserve',
                'consume',
                'debit',
                'release',
                'finalize',
                'refund',
                'adjustment'
            )
        ),
    CONSTRAINT chk_wallet_events_snapshot
        CHECK (
            balance_after_micros >= 0
            AND reserved_after_micros >= 0
            AND reserved_after_micros <= balance_after_micros
        ),
    CONSTRAINT chk_wallet_events_not_empty
        CHECK (
            balance_delta_micros <> 0
            OR reserved_delta_micros <> 0
            OR event_type = 'finalize'
        )
);

CREATE UNIQUE INDEX uq_wallet_events_charge_sequence
    ON wallet_events (charge_id, charge_sequence)
    WHERE charge_id IS NOT NULL;

CREATE INDEX idx_wallet_events_wallet_version
    ON wallet_events (wallet_id, wallet_version);

CREATE INDEX idx_wallet_events_charge_sequence
    ON wallet_events (charge_id, charge_sequence)
    WHERE charge_id IS NOT NULL;

CREATE FUNCTION reject_wallet_event_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'wallet events are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER wallet_events_immutable
BEFORE UPDATE OR DELETE ON wallet_events
FOR EACH ROW EXECUTE FUNCTION reject_wallet_event_mutation();

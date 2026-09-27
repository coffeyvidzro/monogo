-- Billing keeps durable financial history in PostgreSQL while the realtime OCS
-- may authorize and mutate hot wallet state in Redis. The mutable wallet and
-- charge rows below are durable projections for recovery and reconciliation;
-- they must not be used as the realtime authorization path.
--
-- Money is stored in integer micros (1 USD = 1,000,000 micros). No billing
-- decision may depend on floating-point arithmetic.

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    balance_micros BIGINT NOT NULL DEFAULT 0,
    reserved_micros BIGINT NOT NULL DEFAULT 0,
    ocs_version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wallets_organization_currency
        UNIQUE (organization_id, currency),
    CONSTRAINT uq_wallets_id_organization
        UNIQUE (id, organization_id),
    CONSTRAINT chk_wallets_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_wallets_status
        CHECK (status IN ('active', 'frozen', 'closed')),
    CONSTRAINT chk_wallets_balance_nonnegative
        CHECK (balance_micros >= 0),
    CONSTRAINT chk_wallets_reserved_nonnegative
        CHECK (reserved_micros >= 0),
    CONSTRAINT chk_wallets_reserved_within_balance
        CHECK (reserved_micros <= balance_micros),
    CONSTRAINT chk_wallets_ocs_version
        CHECK (ocs_version >= 0),
    CONSTRAINT chk_wallets_closed_without_reservation
        CHECK (status <> 'closed' OR reserved_micros = 0)
);

CREATE INDEX idx_wallets_organization
    ON wallets (organization_id, created_at DESC);

CREATE TRIGGER set_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- A charge represents one financially authorized resource operation. It is
-- provider-neutral: calls, AI sessions, messages, and number operations use the
-- same boundary. The pricing snapshot explains the quote that created the
-- charge; the OCS consumes only the resulting integer-micros amounts.
CREATE TABLE charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    wallet_id UUID NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id UUID NOT NULL,
    charging_mode TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    idempotency_key TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    pricing_snapshot JSONB NOT NULL DEFAULT '{}'::jsonb,
    status TEXT NOT NULL DEFAULT 'pending',
    authorized_micros BIGINT NOT NULL DEFAULT 0,
    consumed_micros BIGINT NOT NULL DEFAULT 0,
    reserved_micros BIGINT NOT NULL DEFAULT 0,
    closed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_charges_wallet_organization
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT uq_charges_id_organization_wallet
        UNIQUE (id, organization_id, wallet_id),
    CONSTRAINT uq_charges_organization_idempotency
        UNIQUE (organization_id, idempotency_key),
    CONSTRAINT chk_charges_resource_type
        CHECK (resource_type ~ '^[a-z][a-z0-9_]{0,63}$'),
    CONSTRAINT chk_charges_charging_mode
        CHECK (charging_mode IN ('rolling', 'discrete')),
    CONSTRAINT chk_charges_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_charges_idempotency_key
        CHECK (
            length(idempotency_key) BETWEEN 1 AND 255
            AND idempotency_key = btrim(idempotency_key)
        ),
    CONSTRAINT chk_charges_request_hash
        CHECK (request_hash ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_charges_pricing_snapshot
        CHECK (jsonb_typeof(pricing_snapshot) = 'object'),
    CONSTRAINT chk_charges_status
        CHECK (status IN ('pending', 'active', 'completed', 'failed', 'cancelled')),
    CONSTRAINT chk_charges_amounts_nonnegative
        CHECK (
            authorized_micros >= 0
            AND consumed_micros >= 0
            AND reserved_micros >= 0
        ),
    CONSTRAINT chk_charges_authorization_covers_exposure
        CHECK (consumed_micros + reserved_micros <= authorized_micros),
    CONSTRAINT chk_charges_active_authorized
        CHECK (status <> 'active' OR authorized_micros > 0),
    CONSTRAINT chk_charges_terminal_releases_reservation
        CHECK (
            status NOT IN ('completed', 'failed', 'cancelled')
            OR reserved_micros = 0
        ),
    CONSTRAINT chk_charges_closed_at
        CHECK (
            (status IN ('pending', 'active') AND closed_at IS NULL)
            OR
            (status IN ('completed', 'failed', 'cancelled') AND closed_at IS NOT NULL)
        )
);

CREATE INDEX idx_charges_organization_created
    ON charges (organization_id, created_at DESC);

CREATE INDEX idx_charges_wallet_status
    ON charges (wallet_id, status, created_at DESC);

CREATE INDEX idx_charges_resource
    ON charges (organization_id, resource_type, resource_id, created_at DESC);

CREATE INDEX idx_charges_active
    ON charges (wallet_id, created_at)
    WHERE status = 'active';

CREATE TRIGGER set_charges_updated_at
BEFORE UPDATE ON charges
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Every successful atomic OCS mutation is mirrored as one immutable event.
-- wallet_version is monotonic per wallet and lets PostgreSQL detect duplicate,
-- missing, or out-of-order Redis stream deliveries. charge_sequence provides
-- the same replay boundary within one long-running charge.
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
    charge_authorized_after_micros BIGINT,
    charge_consumed_after_micros BIGINT,
    charge_reserved_after_micros BIGINT,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_events_wallet_organization
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_events_charge_organization_wallet
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
                'tick',
                'debit',
                'release',
                'finalize',
                'refund',
                'adjustment'
            )
        ),
    CONSTRAINT chk_wallet_events_wallet_snapshot
        CHECK (
            balance_after_micros >= 0
            AND reserved_after_micros >= 0
            AND reserved_after_micros <= balance_after_micros
        ),
    CONSTRAINT chk_wallet_events_charge_snapshot
        CHECK (
            (
                charge_id IS NULL
                AND charge_authorized_after_micros IS NULL
                AND charge_consumed_after_micros IS NULL
                AND charge_reserved_after_micros IS NULL
            )
            OR
            (
                charge_id IS NOT NULL
                AND charge_authorized_after_micros IS NOT NULL
                AND charge_consumed_after_micros IS NOT NULL
                AND charge_reserved_after_micros IS NOT NULL
                AND charge_authorized_after_micros >= 0
                AND charge_consumed_after_micros >= 0
                AND charge_reserved_after_micros >= 0
                AND charge_consumed_after_micros + charge_reserved_after_micros
                    <= charge_authorized_after_micros
            )
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
    WHERE charge_id IS NOT NULL AND charge_sequence IS NOT NULL;

CREATE INDEX idx_wallet_events_wallet_occurred
    ON wallet_events (wallet_id, occurred_at, wallet_version);

CREATE INDEX idx_wallet_events_charge_occurred
    ON wallet_events (charge_id, occurred_at, charge_sequence)
    WHERE charge_id IS NOT NULL;

-- Settled money movements form the immutable financial ledger. Reservations do
-- not create ledger entries because they do not change settled wallet balance.
-- Refunds and corrections are new entries; historical entries are never edited.
CREATE TABLE wallet_ledger_entries (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    wallet_event_id UUID NOT NULL UNIQUE,
    wallet_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    charge_id UUID,
    direction TEXT NOT NULL,
    reason TEXT NOT NULL,
    amount_micros BIGINT NOT NULL,
    balance_after_micros BIGINT NOT NULL,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT fk_wallet_ledger_wallet_organization
        FOREIGN KEY (wallet_id, organization_id)
        REFERENCES wallets (id, organization_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_charge_organization_wallet
        FOREIGN KEY (charge_id, organization_id, wallet_id)
        REFERENCES charges (id, organization_id, wallet_id) ON DELETE RESTRICT,
    CONSTRAINT fk_wallet_ledger_event_organization_wallet
        FOREIGN KEY (wallet_event_id, organization_id, wallet_id)
        REFERENCES wallet_events (id, organization_id, wallet_id) ON DELETE RESTRICT,
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

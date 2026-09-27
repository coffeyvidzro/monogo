-- A charge is one financially authorized resource operation. The same
-- primitive can later back managed calls, AI sessions, messages, or number
-- purchases without making the billing domain provider-specific.

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

    CONSTRAINT fk_charges_wallet
        FOREIGN KEY (wallet_id, organization_id, currency)
        REFERENCES wallets (id, organization_id, currency) ON DELETE RESTRICT,
    CONSTRAINT uq_charges_id_organization_wallet
        UNIQUE (id, organization_id, wallet_id),
    CONSTRAINT uq_charges_resource
        UNIQUE (organization_id, resource_type, resource_id),
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
    CONSTRAINT chk_charges_amounts
        CHECK (
            authorized_micros >= 0
            AND consumed_micros >= 0
            AND reserved_micros >= 0
            AND consumed_micros + reserved_micros <= authorized_micros
        ),
    CONSTRAINT chk_charges_active_authorized
        CHECK (status <> 'active' OR authorized_micros > 0),
    CONSTRAINT chk_charges_terminal_reservation
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

CREATE INDEX idx_charges_active
    ON charges (wallet_id, created_at)
    WHERE status = 'active';

CREATE TRIGGER set_charges_updated_at
BEFORE UPDATE ON charges
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

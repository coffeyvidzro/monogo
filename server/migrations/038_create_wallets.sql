-- Prepaid wallet projection for Leamout's Online Charging System (OCS).
-- Realtime authorization will live in Redis; PostgreSQL remains the durable
-- recovery and reconciliation boundary.
--
-- Money is always integer micros: 1 USD = 1,000,000 micros.

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
    CONSTRAINT uq_wallets_id_organization_currency
        UNIQUE (id, organization_id, currency),
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

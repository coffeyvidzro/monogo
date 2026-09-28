-- One prepaid wallet per organization and currency.
-- Money uses integer micros: 1 USD = 1,000,000 micros.

CREATE TABLE wallets (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    currency CHAR(3) NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    balance_micros BIGINT NOT NULL DEFAULT 0,
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
        CHECK (balance_micros >= 0)
);

CREATE INDEX idx_wallets_organization
    ON wallets (organization_id, created_at DESC);

CREATE TRIGGER set_wallets_updated_at
BEFORE UPDATE ON wallets
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

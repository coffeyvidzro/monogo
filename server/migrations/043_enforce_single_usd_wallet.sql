-- An organization has exactly one prepaid wallet and Leamout wallets are USD-only.
ALTER TABLE wallets
    DROP CONSTRAINT uq_wallets_organization_currency,
    DROP CONSTRAINT chk_wallets_currency,
    ALTER COLUMN currency SET DEFAULT 'USD',
    ADD CONSTRAINT uq_wallets_organization
        UNIQUE (organization_id),
    ADD CONSTRAINT chk_wallets_currency
        CHECK (currency = 'USD');

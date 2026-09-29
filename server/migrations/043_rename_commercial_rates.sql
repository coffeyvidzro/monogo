-- Preserve historical migration versions and rename the commercial rate
-- tables forward from the schema already shipped on main.

ALTER TABLE provider_rates
    RENAME TO provider_voice_rates;

ALTER INDEX idx_provider_rates_lookup
    RENAME TO idx_provider_voice_rates_lookup;

ALTER TABLE carrier_rates
    RENAME TO voice_rates;

ALTER TABLE voice_rates
    RENAME CONSTRAINT chk_carrier_rates_destination_prefix
    TO chk_voice_rates_destination_prefix;

ALTER TABLE voice_rates
    RENAME CONSTRAINT chk_carrier_rates_direction
    TO chk_voice_rates_direction;

ALTER TABLE voice_rates
    RENAME CONSTRAINT chk_carrier_rates_currency
    TO chk_voice_rates_currency;

ALTER TABLE voice_rates
    RENAME CONSTRAINT chk_carrier_rates_rate
    TO chk_voice_rates_rate;

ALTER TABLE voice_rates
    RENAME CONSTRAINT chk_carrier_rates_effective_window
    TO chk_voice_rates_effective_window;

ALTER INDEX uq_carrier_rates_global
    RENAME TO uq_voice_rates_global;

ALTER INDEX uq_carrier_rates_organization
    RENAME TO uq_voice_rates_organization;

ALTER INDEX idx_carrier_rates_global_lookup
    RENAME TO idx_voice_rates_global_lookup;

ALTER INDEX idx_carrier_rates_organization_lookup
    RENAME TO idx_voice_rates_organization_lookup;

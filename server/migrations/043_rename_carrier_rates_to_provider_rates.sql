-- The existing carrier_rates table stores upstream carrier cost used by
-- managed route selection. Rename it so carrier_rates can represent the
-- customer-facing retail rate Leamout charges.
ALTER TABLE carrier_rates RENAME TO provider_rates;

ALTER INDEX idx_carrier_rates_lookup
RENAME TO idx_provider_rates_lookup;

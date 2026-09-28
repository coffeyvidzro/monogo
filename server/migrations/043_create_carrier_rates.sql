-- Customer-facing managed carrier pricing. This is what Leamout charges
-- customers and is separate from provider_rates, which is supplier pricing.

CREATE TABLE carrier_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
    destination_prefix TEXT NOT NULL,
    direction TEXT NOT NULL DEFAULT 'outbound',
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    rate_micros BIGINT NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_carrier_rates_destination_prefix
        CHECK (destination_prefix ~ '^[1-9][0-9]{0,14}$'),
    CONSTRAINT chk_carrier_rates_direction
        CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT chk_carrier_rates_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_carrier_rates_rate
        CHECK (rate_micros >= 0),
    CONSTRAINT chk_carrier_rates_effective_window
        CHECK (expires_at IS NULL OR expires_at > effective_at)
);

CREATE UNIQUE INDEX uq_carrier_rates_global
    ON carrier_rates (
        destination_prefix,
        direction,
        currency,
        effective_at
    )
    WHERE organization_id IS NULL;

CREATE UNIQUE INDEX uq_carrier_rates_organization
    ON carrier_rates (
        organization_id,
        destination_prefix,
        direction,
        currency,
        effective_at
    )
    WHERE organization_id IS NOT NULL;

CREATE INDEX idx_carrier_rates_global_lookup
    ON carrier_rates (
        destination_prefix,
        direction,
        currency,
        effective_at DESC
    )
    WHERE organization_id IS NULL;

CREATE INDEX idx_carrier_rates_organization_lookup
    ON carrier_rates (
        organization_id,
        destination_prefix,
        direction,
        currency,
        effective_at DESC
    )
    WHERE organization_id IS NOT NULL;

-- Customer-facing per-started-minute managed voice pricing. Longest-prefix
-- resolution is separate from provider_rates, which is supplier pricing.

CREATE TABLE voice_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
    destination_prefix TEXT NOT NULL,
    direction TEXT NOT NULL DEFAULT 'outbound',
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    rate_micros BIGINT NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_voice_rates_destination_prefix
        CHECK (destination_prefix ~ '^[1-9][0-9]{0,14}$'),
    CONSTRAINT chk_voice_rates_direction
        CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT chk_voice_rates_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_voice_rates_rate
        CHECK (rate_micros > 0),
    CONSTRAINT chk_voice_rates_effective_window
        CHECK (expires_at IS NULL OR expires_at > effective_at)
);

CREATE UNIQUE INDEX uq_voice_rates_global
    ON voice_rates (
        destination_prefix,
        direction,
        currency,
        effective_at
    )
    WHERE organization_id IS NULL;

CREATE UNIQUE INDEX uq_voice_rates_organization
    ON voice_rates (
        organization_id,
        destination_prefix,
        direction,
        currency,
        effective_at
    )
    WHERE organization_id IS NOT NULL;

CREATE INDEX idx_voice_rates_global_lookup
    ON voice_rates (
        destination_prefix,
        direction,
        currency,
        effective_at DESC
    )
    WHERE organization_id IS NULL;

CREATE INDEX idx_voice_rates_organization_lookup
    ON voice_rates (
        organization_id,
        destination_prefix,
        direction,
        currency,
        effective_at DESC
    )
    WHERE organization_id IS NOT NULL;

-- Customer-facing fixed-unit prices for managed non-voice products. A selector
-- is either an ISO 3166-1 alpha-2 country code or the global '*' fallback.
CREATE TABLE product_rates (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE RESTRICT,
    product TEXT NOT NULL,
    selector TEXT NOT NULL DEFAULT '*',
    currency CHAR(3) NOT NULL DEFAULT 'USD',
    rate_micros BIGINT NOT NULL,
    effective_at TIMESTAMPTZ NOT NULL,
    expires_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_product_rates_product CHECK (product IN (
        'sms_outbound',
        'whatsapp_outbound',
        'number_purchase',
        'number_renewal'
    )),
    CONSTRAINT chk_product_rates_selector
        CHECK (selector = '*' OR selector ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_product_rates_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_product_rates_rate CHECK (rate_micros > 0),
    CONSTRAINT chk_product_rates_window
        CHECK (expires_at IS NULL OR expires_at > effective_at)
);

CREATE UNIQUE INDEX uq_product_rates_global
    ON product_rates (product, selector, currency, effective_at)
    WHERE organization_id IS NULL;

CREATE UNIQUE INDEX uq_product_rates_organization
    ON product_rates (
        organization_id,
        product,
        selector,
        currency,
        effective_at
    )
    WHERE organization_id IS NOT NULL;

CREATE INDEX idx_product_rates_resolve
    ON product_rates (product, selector, currency, effective_at DESC);

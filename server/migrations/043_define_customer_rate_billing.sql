-- Customer carrier rates are quoted per minute. Minimum duration and billing
-- increment are explicit commercial terms rather than hidden arithmetic.
ALTER TABLE carrier_rates
    ADD COLUMN billing_unit TEXT NOT NULL DEFAULT 'minute',
    ADD COLUMN billing_increment_seconds INTEGER NOT NULL DEFAULT 60,
    ADD COLUMN minimum_duration_seconds INTEGER NOT NULL DEFAULT 60,
    ADD CONSTRAINT chk_carrier_rates_billing_unit CHECK (billing_unit = 'minute'),
    ADD CONSTRAINT chk_carrier_rates_billing_increment CHECK (billing_increment_seconds > 0),
    ADD CONSTRAINT chk_carrier_rates_minimum_duration CHECK (minimum_duration_seconds >= 0);

-- Managed calls snapshot the selected customer price before origination.
ALTER TABLE calls
    ADD COLUMN customer_carrier_rate_id UUID REFERENCES carrier_rates(id) ON DELETE RESTRICT,
    ADD COLUMN customer_rate_currency CHAR(3),
    ADD COLUMN customer_rate_micros BIGINT,
    ADD COLUMN customer_rate_billing_unit TEXT,
    ADD COLUMN customer_rate_billing_increment_seconds INTEGER,
    ADD COLUMN customer_rate_minimum_duration_seconds INTEGER,
    ADD COLUMN commercial_authorized_at TIMESTAMPTZ,
    ADD CONSTRAINT chk_calls_customer_rate_snapshot CHECK (
        (customer_carrier_rate_id IS NULL
            AND customer_rate_currency IS NULL
            AND customer_rate_micros IS NULL
            AND customer_rate_billing_unit IS NULL
            AND customer_rate_billing_increment_seconds IS NULL
            AND customer_rate_minimum_duration_seconds IS NULL
            AND commercial_authorized_at IS NULL)
        OR
        (customer_carrier_rate_id IS NOT NULL
            AND customer_rate_currency IS NOT NULL
            AND customer_rate_micros >= 0
            AND customer_rate_billing_unit = 'minute'
            AND customer_rate_billing_increment_seconds > 0
            AND customer_rate_minimum_duration_seconds >= 0
            AND commercial_authorized_at IS NOT NULL)
    );

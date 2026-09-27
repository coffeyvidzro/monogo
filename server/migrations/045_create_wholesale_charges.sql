-- Wholesale charges are Leamout's supplier cost for managed carrier usage.
-- The upstream provider CDR remains the immutable evidence for each charge.

CREATE TABLE wholesale_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_cdr_id UUID NOT NULL REFERENCES provider_cdrs(id) ON DELETE RESTRICT,

    currency CHAR(3) NOT NULL,
    rate_micros BIGINT NOT NULL,
    billable_seconds BIGINT NOT NULL,
    amount_micros BIGINT NOT NULL,

    rated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_wholesale_charges_provider_cdr
        UNIQUE (provider_cdr_id),
    CONSTRAINT chk_wholesale_charges_currency
        CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_wholesale_charges_rate
        CHECK (rate_micros >= 0),
    CONSTRAINT chk_wholesale_charges_billable
        CHECK (billable_seconds >= 0),
    CONSTRAINT chk_wholesale_charges_amount
        CHECK (amount_micros >= 0)
);

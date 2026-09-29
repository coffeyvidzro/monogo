-- Managed-number renewal periods are explicit so retries cannot charge a
-- customer twice and failed payments remain visible for recovery.
ALTER TABLE phone_numbers
    ADD COLUMN next_renewal_at TIMESTAMPTZ;

ALTER TABLE phone_numbers
    ADD CONSTRAINT chk_phone_numbers_managed_renewal
    CHECK (
        (provisioning_mode = 'managed' AND status <> 'released')
        OR next_renewal_at IS NULL
    );

CREATE TABLE managed_number_renewals (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    phone_number_id UUID NOT NULL REFERENCES phone_numbers(id) ON DELETE RESTRICT,
    period_start TIMESTAMPTZ NOT NULL,
    period_end TIMESTAMPTZ NOT NULL,
    operation_id UUID NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    amount_micros BIGINT,
    currency CHAR(3),
    attempt_count INTEGER NOT NULL DEFAULT 0,
    next_attempt_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    paid_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_managed_number_renewals_period UNIQUE (phone_number_id, period_start),
    CONSTRAINT uq_managed_number_renewals_operation UNIQUE (operation_id),
    CONSTRAINT chk_managed_number_renewals_period CHECK (period_end > period_start),
    CONSTRAINT chk_managed_number_renewals_status
        CHECK (status IN ('pending', 'processing', 'paid', 'payment_failed')),
    CONSTRAINT chk_managed_number_renewals_amount
        CHECK (amount_micros IS NULL OR amount_micros > 0),
    CONSTRAINT chk_managed_number_renewals_currency
        CHECK (currency IS NULL OR currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_managed_number_renewals_attempts CHECK (attempt_count >= 0),
    CONSTRAINT chk_managed_number_renewals_paid
        CHECK ((status = 'paid') = (paid_at IS NOT NULL))
);

CREATE INDEX idx_managed_number_renewals_due
    ON managed_number_renewals (next_attempt_at, period_start)
    WHERE status IN ('pending', 'payment_failed');

CREATE TRIGGER set_managed_number_renewals_updated_at
BEFORE UPDATE ON managed_number_renewals
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Margin reporting joins captured customer revenue to reconciled supplier charges.
CREATE VIEW managed_margin_entries AS
SELECT
    ledger.operation_id,
    ledger.organization_id,
    ledger.reference_type,
    ledger.reference_id,
    ledger.currency,
    ledger.amount_micros AS revenue_micros,
    COUNT(charge.id) FILTER (WHERE charge.currency = ledger.currency)::BIGINT
        AS provider_charge_record_count,
    COALESCE(SUM(charge.amount_micros) FILTER (WHERE charge.currency = ledger.currency), 0)::BIGINT
        AS provider_charge_micros,
    ledger.amount_micros
        - COALESCE(SUM(charge.amount_micros) FILTER (WHERE charge.currency = ledger.currency), 0)::BIGINT
        AS gross_profit_micros,
    ledger.occurred_at
FROM (
    SELECT entries.*, wallets.currency
    FROM wallet_ledger_entries AS entries
    JOIN wallets ON wallets.id = entries.wallet_id
    WHERE entries.direction = 'debit'
) AS ledger
LEFT JOIN provider_charges AS charge ON charge.operation_id = ledger.operation_id
GROUP BY
    ledger.operation_id,
    ledger.organization_id,
    ledger.reference_type,
    ledger.reference_id,
    ledger.currency,
    ledger.amount_micros,
    ledger.occurred_at;

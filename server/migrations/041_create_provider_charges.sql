-- Actual immutable supplier charges for every managed product. Provider
-- evidence may be recorded before it is reconciled to customer revenue.
CREATE TABLE provider_charges (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    provider_cdr_id UUID REFERENCES provider_cdrs(id) ON DELETE RESTRICT,
    operation_id UUID REFERENCES wallet_ledger_entries(operation_id) ON DELETE RESTRICT,
    provider_record_type TEXT NOT NULL,
    provider_record_id TEXT NOT NULL,
    product TEXT NOT NULL,
    currency CHAR(3) NOT NULL,
    rate_micros BIGINT,
    billable_seconds BIGINT,
    amount_micros BIGINT NOT NULL,
    incurred_at TIMESTAMPTZ NOT NULL,
    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    recorded_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_provider_charges_record
        UNIQUE (provider_id, provider_record_type, provider_record_id),
    CONSTRAINT uq_provider_charges_cdr UNIQUE (provider_cdr_id),
    CONSTRAINT chk_provider_charges_record_type
        CHECK (provider_record_type IN ('voice_cdr', 'order', 'message', 'invoice_item')),
    CONSTRAINT chk_provider_charges_record_id
        CHECK (length(btrim(provider_record_id)) BETWEEN 1 AND 255),
    CONSTRAINT chk_provider_charges_product
        CHECK (product IN ('voice', 'sms_outbound', 'whatsapp_outbound',
                           'number_purchase', 'number_renewal')),
    CONSTRAINT chk_provider_charges_currency CHECK (currency ~ '^[A-Z]{3}$'),
    CONSTRAINT chk_provider_charges_rate CHECK (rate_micros IS NULL OR rate_micros >= 0),
    CONSTRAINT chk_provider_charges_billable
        CHECK (billable_seconds IS NULL OR billable_seconds >= 0),
    CONSTRAINT chk_provider_charges_amount CHECK (amount_micros >= 0),
    CONSTRAINT chk_provider_charges_payload CHECK (jsonb_typeof(raw_payload) = 'object'),
    CONSTRAINT chk_provider_charges_voice
        CHECK (
            (product = 'voice' AND provider_record_type = 'voice_cdr'
             AND provider_cdr_id IS NOT NULL AND rate_micros IS NOT NULL
             AND billable_seconds IS NOT NULL)
            OR
            (product <> 'voice' AND provider_cdr_id IS NULL)
        )
);

CREATE INDEX idx_provider_charges_operation
    ON provider_charges (operation_id) WHERE operation_id IS NOT NULL;
CREATE INDEX idx_provider_charges_unreconciled
    ON provider_charges (incurred_at, id) WHERE operation_id IS NULL;
CREATE INDEX idx_provider_charges_incurred ON provider_charges (incurred_at DESC);

CREATE FUNCTION reject_provider_charge_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'provider charges are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER provider_charges_immutable
BEFORE UPDATE OR DELETE ON provider_charges
FOR EACH ROW EXECUTE FUNCTION reject_provider_charge_mutation();

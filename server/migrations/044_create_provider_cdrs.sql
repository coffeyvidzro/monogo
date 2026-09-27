-- Immutable upstream call-detail records received from carrier providers.
-- These records are supplier evidence and are reconciled to Leamout-managed
-- calls before wholesale cost is recorded.

CREATE TABLE provider_cdrs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE RESTRICT,
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE RESTRICT,

    provider_cdr_id TEXT NOT NULL,
    direction TEXT NOT NULL,
    source TEXT,
    destination TEXT,

    started_at TIMESTAMPTZ NOT NULL,
    answered_at TIMESTAMPTZ,
    ended_at TIMESTAMPTZ NOT NULL,
    duration_seconds BIGINT NOT NULL,
    billable_seconds BIGINT NOT NULL,

    raw_payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_provider_cdrs_provider_record
        UNIQUE (provider_id, provider_cdr_id),
    CONSTRAINT chk_provider_cdrs_provider_record
        CHECK (
            length(provider_cdr_id) BETWEEN 1 AND 255
            AND provider_cdr_id = btrim(provider_cdr_id)
        ),
    CONSTRAINT chk_provider_cdrs_direction
        CHECK (direction IN ('inbound', 'outbound')),
    CONSTRAINT chk_provider_cdrs_timestamps
        CHECK (
            ended_at >= started_at
            AND (answered_at IS NULL OR answered_at >= started_at)
            AND (answered_at IS NULL OR ended_at >= answered_at)
        ),
    CONSTRAINT chk_provider_cdrs_duration
        CHECK (duration_seconds >= 0),
    CONSTRAINT chk_provider_cdrs_billable
        CHECK (billable_seconds >= 0),
    CONSTRAINT chk_provider_cdrs_raw_payload
        CHECK (jsonb_typeof(raw_payload) = 'object')
);

CREATE INDEX idx_provider_cdrs_call
    ON provider_cdrs (call_id);

CREATE INDEX idx_provider_cdrs_provider_started
    ON provider_cdrs (provider_id, started_at DESC);

CREATE FUNCTION reject_provider_cdr_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'provider CDRs are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER provider_cdrs_immutable
BEFORE UPDATE OR DELETE ON provider_cdrs
FOR EACH ROW EXECUTE FUNCTION reject_provider_cdr_mutation();

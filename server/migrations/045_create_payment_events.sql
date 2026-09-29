CREATE TABLE payment_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL REFERENCES payments(id) ON DELETE RESTRICT,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    failure_code TEXT,
    payload JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_payment_events_provider_event
        UNIQUE (provider, provider_event_id),
    CONSTRAINT chk_payment_events_type
        CHECK (event_type IN ('succeeded', 'failed', 'cancelled')),
    CONSTRAINT chk_payment_events_failure
        CHECK (
            (event_type = 'failed' AND failure_code IS NOT NULL)
            OR
            (event_type <> 'failed' AND failure_code IS NULL)
        )
);

CREATE INDEX idx_payment_events_payment_created
    ON payment_events (payment_id, occurred_at DESC, id DESC);

CREATE FUNCTION reject_payment_event_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'payment events are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER payment_events_immutable
BEFORE UPDATE OR DELETE ON payment_events
FOR EACH ROW EXECUTE FUNCTION reject_payment_event_mutation();

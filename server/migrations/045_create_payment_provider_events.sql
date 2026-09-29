CREATE TABLE payment_provider_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    payment_id UUID NOT NULL,
    organization_id UUID NOT NULL,
    provider TEXT NOT NULL,
    provider_event_id TEXT NOT NULL,
    event_type TEXT NOT NULL,
    payload_sha256 TEXT NOT NULL,
    payload JSONB NOT NULL,
    received_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    processed_at TIMESTAMPTZ,

    CONSTRAINT fk_payment_provider_events_payment_organization
        FOREIGN KEY (payment_id, organization_id)
        REFERENCES payments (id, organization_id)
        ON DELETE RESTRICT,
    CONSTRAINT uq_payment_provider_events_identity
        UNIQUE (provider, provider_event_id),
    CONSTRAINT chk_payment_provider_events_provider
        CHECK (provider IN ('stripe', 'paystack')),
    CONSTRAINT chk_payment_provider_events_id
        CHECK (length(btrim(provider_event_id)) > 0),
    CONSTRAINT chk_payment_provider_events_type
        CHECK (length(btrim(event_type)) > 0),
    CONSTRAINT chk_payment_provider_events_hash
        CHECK (payload_sha256 ~ '^[0-9a-f]{64}$'),
    CONSTRAINT chk_payment_provider_events_payload
        CHECK (jsonb_typeof(payload) = 'object')
);

CREATE INDEX idx_payment_provider_events_unprocessed
    ON payment_provider_events (received_at, id)
    WHERE processed_at IS NULL;

CREATE FUNCTION enforce_payment_provider_event_immutability()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'payment provider events are immutable'
            USING ERRCODE = '23514';
    END IF;

    IF OLD.processed_at IS NOT NULL
        OR NEW.processed_at IS NULL
        OR NEW.id IS DISTINCT FROM OLD.id
        OR NEW.payment_id IS DISTINCT FROM OLD.payment_id
        OR NEW.organization_id IS DISTINCT FROM OLD.organization_id
        OR NEW.provider IS DISTINCT FROM OLD.provider
        OR NEW.provider_event_id IS DISTINCT FROM OLD.provider_event_id
        OR NEW.event_type IS DISTINCT FROM OLD.event_type
        OR NEW.payload_sha256 IS DISTINCT FROM OLD.payload_sha256
        OR NEW.payload IS DISTINCT FROM OLD.payload
        OR NEW.received_at IS DISTINCT FROM OLD.received_at
        OR NEW.created_at IS DISTINCT FROM OLD.created_at
    THEN
        RAISE EXCEPTION 'payment provider events are immutable'
            USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$;

CREATE TRIGGER payment_provider_events_immutable
BEFORE UPDATE OR DELETE ON payment_provider_events
FOR EACH ROW EXECUTE FUNCTION enforce_payment_provider_event_immutability();

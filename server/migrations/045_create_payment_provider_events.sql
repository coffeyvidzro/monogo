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

CREATE INDEX idx_payment_provider_events_payment_received
    ON payment_provider_events (payment_id, received_at DESC, id DESC);

CREATE FUNCTION protect_payment_provider_event()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $
BEGIN
    IF TG_OP = 'DELETE' THEN
        RAISE EXCEPTION 'payment provider events are immutable' USING ERRCODE = '23514';
    END IF;

    IF OLD.id IS DISTINCT FROM NEW.id
        OR OLD.payment_id IS DISTINCT FROM NEW.payment_id
        OR OLD.organization_id IS DISTINCT FROM NEW.organization_id
        OR OLD.provider IS DISTINCT FROM NEW.provider
        OR OLD.provider_event_id IS DISTINCT FROM NEW.provider_event_id
        OR OLD.event_type IS DISTINCT FROM NEW.event_type
        OR OLD.payload_sha256 IS DISTINCT FROM NEW.payload_sha256
        OR OLD.payload IS DISTINCT FROM NEW.payload
        OR OLD.received_at IS DISTINCT FROM NEW.received_at
        OR OLD.processed_at IS NOT NULL
        OR NEW.processed_at IS NULL
    THEN
        RAISE EXCEPTION 'payment provider event payload is immutable' USING ERRCODE = '23514';
    END IF;

    RETURN NEW;
END;
$;

CREATE TRIGGER payment_provider_events_protected
BEFORE UPDATE OR DELETE ON payment_provider_events
FOR EACH ROW EXECUTE FUNCTION protect_payment_provider_event();

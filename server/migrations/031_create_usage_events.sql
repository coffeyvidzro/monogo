-- The proposed usage_events.meter_id foreign key requires a meter registry.
-- Meter definitions are organization-scoped and contain no price or billing rule.
CREATE TABLE IF NOT EXISTS meters (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    name TEXT NOT NULL,
    unit TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_meters_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_meters_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_meters_name CHECK (name ~ '^[a-z0-9]+(?:_[a-z0-9]+)*$'),
    CONSTRAINT chk_meters_unit CHECK (length(btrim(unit)) > 0),
    CONSTRAINT chk_meters_status CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_meters_organization_status
    ON meters (organization_id, status);

CREATE TRIGGER set_meters_updated_at
BEFORE UPDATE ON meters
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

COMMENT ON TABLE meters IS
    'Organization-scoped measurement definitions; neither prices nor makes usage billable.';

CREATE TABLE IF NOT EXISTS usage_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    meter_id UUID NOT NULL REFERENCES meters(id) ON DELETE RESTRICT,
    quantity BIGINT NOT NULL,
    source_type TEXT NOT NULL,
    source_id TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    dimensions JSONB NOT NULL DEFAULT '{}'::jsonb,
    occurred_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_usage_events_organization_idempotency
        UNIQUE (organization_id, idempotency_key),
    CONSTRAINT fk_usage_events_organization_meter
        FOREIGN KEY (meter_id, organization_id) REFERENCES meters (id, organization_id),
    CONSTRAINT chk_usage_events_quantity CHECK (quantity > 0),
    CONSTRAINT chk_usage_events_source_type CHECK (
        source_type ~ '^[a-z0-9]+(?:_[a-z0-9]+)*$'
    ),
    CONSTRAINT chk_usage_events_source_id CHECK (length(btrim(source_id)) > 0),
    CONSTRAINT chk_usage_events_idempotency_key CHECK (
        length(btrim(idempotency_key)) > 0
        AND length(idempotency_key) <= 255
        AND idempotency_key = btrim(idempotency_key)
    ),
    CONSTRAINT chk_usage_events_dimensions_object CHECK (
        jsonb_typeof(dimensions) = 'object'
    )
);

COMMENT ON TABLE usage_events IS
    'Immutable usage observations. Recording usage does not by itself make that usage billable.';

CREATE INDEX IF NOT EXISTS idx_usage_events_organization_meter_occurred
    ON usage_events (organization_id, meter_id, occurred_at);

CREATE INDEX IF NOT EXISTS idx_usage_events_source
    ON usage_events (source_type, source_id);

-- A correction must be a distinct observation; accepted records cannot be
-- rewritten or deleted.
CREATE FUNCTION reject_usage_event_mutation()
RETURNS TRIGGER LANGUAGE plpgsql AS $$
BEGIN
    RAISE EXCEPTION 'usage events are immutable' USING ERRCODE = '23514';
END;
$$;

CREATE TRIGGER usage_events_immutable
BEFORE UPDATE OR DELETE ON usage_events
FOR EACH ROW EXECUTE FUNCTION reject_usage_event_mutation();

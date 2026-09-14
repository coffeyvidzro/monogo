CREATE TABLE audit_events (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id),
    actor_type TEXT NOT NULL,
    actor_id UUID NOT NULL,
    action TEXT NOT NULL,
    target_type TEXT NOT NULL,
    target_id UUID NOT NULL,
    metadata JSONB NOT NULL DEFAULT '{}'::JSONB,
    occurred_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    CONSTRAINT chk_audit_actor_type CHECK (actor_type IN ('user', 'organization_token')),
    CONSTRAINT chk_audit_action_nonempty CHECK (length(trim(action)) > 0),
    CONSTRAINT chk_audit_target_type_nonempty CHECK (length(trim(target_type)) > 0)
);

CREATE INDEX idx_audit_events_organization_time
    ON audit_events (organization_id, occurred_at DESC, id DESC);

CREATE INDEX idx_audit_events_target
    ON audit_events (organization_id, target_type, target_id, occurred_at DESC);

COMMENT ON TABLE audit_events IS
    'Append-only security and configuration audit history. Metadata must never contain plaintext credentials.';

CREATE FUNCTION reject_audit_event_mutation()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    RAISE EXCEPTION 'audit events are append-only';
END;
$$;

CREATE TRIGGER trg_audit_events_append_only
BEFORE UPDATE OR DELETE ON audit_events
FOR EACH ROW EXECUTE FUNCTION reject_audit_event_mutation();

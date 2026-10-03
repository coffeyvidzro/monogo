CREATE TABLE IF NOT EXISTS retention_policies (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    resource TEXT NOT NULL,
    retention_days INTEGER NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (organization_id, resource),
    CONSTRAINT chk_retention_policies_resource
        CHECK (resource IN ('recordings', 'conversations', 'audit_events')),
    CONSTRAINT chk_retention_policies_days
        CHECK (retention_days BETWEEN 1 AND 3650)
);

CREATE INDEX IF NOT EXISTS idx_retention_policies_enabled
    ON retention_policies (organization_id, resource)
    WHERE enabled;

CREATE TRIGGER set_retention_policies_updated_at
BEFORE UPDATE ON retention_policies
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

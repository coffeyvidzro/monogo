CREATE TABLE IF NOT EXISTS organization_entitlements (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    capability TEXT NOT NULL,
    enabled BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (organization_id, capability),
    CONSTRAINT chk_organization_entitlements_capability
        CHECK (capability IN (
            'sso',
            'scim',
            'advanced_rbac',
            'retention_policies',
            'private_networking'
        ))
);

CREATE INDEX IF NOT EXISTS idx_organization_entitlements_enabled
    ON organization_entitlements (organization_id, capability)
    WHERE enabled;

CREATE TRIGGER set_organization_entitlements_updated_at
BEFORE UPDATE ON organization_entitlements
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

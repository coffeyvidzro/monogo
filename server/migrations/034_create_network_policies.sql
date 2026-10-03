CREATE TABLE IF NOT EXISTS network_policies (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    action TEXT NOT NULL,
    source_cidr CIDR NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_network_policies_name UNIQUE (organization_id, name),
    CONSTRAINT uq_network_policies_rule
        UNIQUE (organization_id, action, source_cidr),
    CONSTRAINT chk_network_policies_name
        CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_network_policies_action
        CHECK (action IN ('allow', 'deny')),
    CONSTRAINT chk_network_policies_status
        CHECK (status IN ('active', 'disabled'))
);

CREATE INDEX IF NOT EXISTS idx_network_policies_organization
    ON network_policies (organization_id, created_at DESC);

CREATE TRIGGER set_network_policies_updated_at
BEFORE UPDATE ON network_policies
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

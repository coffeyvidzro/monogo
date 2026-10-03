CREATE TABLE IF NOT EXISTS runtimes (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',
    version TEXT,
    region TEXT,
    capabilities TEXT[] NOT NULL DEFAULT '{}',
    capacity INTEGER NOT NULL DEFAULT 0,
    active_sessions INTEGER NOT NULL DEFAULT 0,
    last_seen_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_runtimes_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_runtimes_organization_name UNIQUE (organization_id, name),
    CONSTRAINT chk_runtimes_name CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_runtimes_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_runtimes_version CHECK (
        version IS NULL OR length(btrim(version)) BETWEEN 1 AND 128
    ),
    CONSTRAINT chk_runtimes_region CHECK (
        region IS NULL OR length(btrim(region)) BETWEEN 1 AND 128
    ),
    CONSTRAINT chk_runtimes_capacity CHECK (capacity >= 0),
    CONSTRAINT chk_runtimes_active_sessions CHECK (
        active_sessions >= 0 AND active_sessions <= capacity
    )
);

CREATE INDEX IF NOT EXISTS idx_runtimes_organization_status
    ON runtimes (organization_id, status, created_at ASC);

CREATE INDEX IF NOT EXISTS idx_runtimes_last_seen
    ON runtimes (organization_id, last_seen_at DESC);

CREATE TRIGGER set_runtimes_updated_at
BEFORE UPDATE ON runtimes
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

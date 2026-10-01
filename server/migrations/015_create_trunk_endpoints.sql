CREATE TABLE IF NOT EXISTS trunk_endpoints (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    trunk_id UUID NOT NULL,
    host TEXT NOT NULL,
    port INTEGER NOT NULL DEFAULT 5060,
    transport TEXT NOT NULL DEFAULT 'udp',
    direction TEXT NOT NULL DEFAULT 'bidirectional',
    priority INTEGER NOT NULL DEFAULT 10,
    weight INTEGER NOT NULL DEFAULT 100,
    enabled BOOLEAN NOT NULL DEFAULT true,
    health_status TEXT NOT NULL DEFAULT 'unknown',
    consecutive_failures INTEGER NOT NULL DEFAULT 0,
    last_checked_at TIMESTAMPTZ,
    last_response_code INTEGER,
    last_latency_ms INTEGER,
    last_error TEXT,
    cooldown_until TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_trunk_endpoints_target
        UNIQUE (trunk_id, host, port, transport, direction),
    CONSTRAINT uq_trunk_endpoints_id_org
        UNIQUE (id, organization_id),
    CONSTRAINT fk_trunk_endpoints_trunk_scope
        FOREIGN KEY (trunk_id, organization_id)
        REFERENCES trunks(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT chk_trunk_endpoints_host
        CHECK (length(btrim(host)) > 0 AND host !~ '\s'),
    CONSTRAINT chk_trunk_endpoints_port
        CHECK (port BETWEEN 1 AND 65535),
    CONSTRAINT chk_trunk_endpoints_transport
        CHECK (transport IN ('udp', 'tcp', 'tls')),
    CONSTRAINT chk_trunk_endpoints_direction
        CHECK (direction IN ('inbound', 'outbound', 'bidirectional')),
    CONSTRAINT chk_trunk_endpoints_priority
        CHECK (priority >= 0),
    CONSTRAINT chk_trunk_endpoints_weight
        CHECK (weight > 0),
    CONSTRAINT chk_trunk_endpoint_health_status
        CHECK (health_status IN ('unknown', 'healthy', 'unhealthy')),
    CONSTRAINT chk_trunk_endpoint_consecutive_failures
        CHECK (consecutive_failures >= 0),
    CONSTRAINT chk_trunk_endpoint_response_code
        CHECK (
            last_response_code IS NULL
            OR last_response_code BETWEEN 100 AND 699
        ),
    CONSTRAINT chk_trunk_endpoint_latency
        CHECK (last_latency_ms IS NULL OR last_latency_ms >= 0)
);

CREATE INDEX IF NOT EXISTS idx_trunk_endpoints_trunk_enabled
    ON trunk_endpoints (trunk_id, enabled, priority);

CREATE INDEX IF NOT EXISTS idx_trunk_endpoints_organization_id
    ON trunk_endpoints (organization_id);

CREATE INDEX IF NOT EXISTS idx_trunk_endpoints_health_probe
    ON trunk_endpoints (
        health_status,
        cooldown_until,
        last_checked_at
    )
    WHERE enabled = true;

CREATE TRIGGER set_trunk_endpoints_updated_at
BEFORE UPDATE ON trunk_endpoints
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

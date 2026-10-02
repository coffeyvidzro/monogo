CREATE TABLE IF NOT EXISTS trunks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    direction TEXT NOT NULL DEFAULT 'bidirectional',
    status TEXT NOT NULL DEFAULT 'active',

    outbound_auth_method TEXT NOT NULL DEFAULT 'none',
    auth_username TEXT,
    auth_realm TEXT,
    auth_secret_ciphertext TEXT,

    inbound_enabled BOOLEAN NOT NULL DEFAULT false,
    inbound_auth_method TEXT NOT NULL DEFAULT 'ip',
    inbound_username TEXT,
    inbound_realm TEXT,
    inbound_secret_ciphertext TEXT,

    max_cps INTEGER NOT NULL DEFAULT 10,
    max_concurrent_calls INTEGER NOT NULL DEFAULT 100,
    codecs TEXT[] NOT NULL DEFAULT '{PCMU,PCMA}',
    supports_video BOOLEAN NOT NULL DEFAULT false,
    supports_fax BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_trunks_org_name UNIQUE (organization_id, name),
    CONSTRAINT uq_trunks_id_org UNIQUE (id, organization_id),
    CONSTRAINT chk_trunks_name CHECK (length(btrim(name)) > 0),
    CONSTRAINT chk_trunks_direction CHECK (
        direction IN ('inbound', 'outbound', 'bidirectional')
    ),
    CONSTRAINT chk_trunks_status CHECK (status IN ('active', 'disabled')),
    CONSTRAINT chk_trunks_outbound_auth_method CHECK (
        outbound_auth_method IN ('none', 'digest')
    ),
    CONSTRAINT chk_trunks_outbound_auth_fields CHECK (
        (
            outbound_auth_method = 'none'
            AND auth_username IS NULL
            AND auth_realm IS NULL
            AND auth_secret_ciphertext IS NULL
        )
        OR (
            outbound_auth_method = 'digest'
            AND auth_username IS NOT NULL
            AND length(btrim(auth_username)) > 0
            AND auth_realm IS NOT NULL
            AND length(btrim(auth_realm)) > 0
            AND auth_secret_ciphertext IS NOT NULL
            AND length(auth_secret_ciphertext) > 0
        )
    ),
    CONSTRAINT chk_trunks_inbound_auth_method CHECK (
        inbound_auth_method IN ('ip', 'digest')
    ),
    CONSTRAINT chk_trunks_inbound_auth_fields CHECK (
        (
            inbound_auth_method = 'ip'
            AND inbound_username IS NULL
            AND inbound_realm IS NULL
            AND inbound_secret_ciphertext IS NULL
        )
        OR (
            inbound_auth_method = 'digest'
            AND inbound_username IS NOT NULL
            AND length(btrim(inbound_username)) > 0
            AND inbound_realm IS NOT NULL
            AND length(btrim(inbound_realm)) > 0
            AND inbound_secret_ciphertext IS NOT NULL
            AND length(inbound_secret_ciphertext) > 0
        )
    ),
    CONSTRAINT chk_trunks_cps CHECK (max_cps > 0),
    CONSTRAINT chk_trunks_concurrent CHECK (max_concurrent_calls > 0)
);

CREATE INDEX IF NOT EXISTS idx_trunks_organization_status
    ON trunks (organization_id, status);

CREATE TABLE IF NOT EXISTS trunk_source_ips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL,
    trunk_id UUID NOT NULL,
    cidr CIDR NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_trunk_source_ips_trunk_cidr UNIQUE (trunk_id, cidr),
    CONSTRAINT fk_trunk_source_ips_scope
        FOREIGN KEY (trunk_id, organization_id)
        REFERENCES trunks(id, organization_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_trunk_source_ips_organization_id
    ON trunk_source_ips (organization_id);

CREATE INDEX IF NOT EXISTS idx_trunk_source_ips_cidr
    ON trunk_source_ips USING gist (cidr inet_ops);

CREATE TRIGGER set_trunks_updated_at
BEFORE UPDATE ON trunks
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

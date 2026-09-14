CREATE TABLE IF NOT EXISTS carrier_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID REFERENCES organizations(id) ON DELETE CASCADE,
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,

    scope TEXT NOT NULL DEFAULT 'organization',
    name TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'active',

    outbound_auth_method TEXT NOT NULL DEFAULT 'none',
    auth_username TEXT,
    auth_secret_ciphertext TEXT,

    inbound_enabled BOOLEAN NOT NULL DEFAULT false,
    inbound_auth_method TEXT NOT NULL DEFAULT 'ip',
    inbound_username TEXT,
    inbound_secret_ciphertext TEXT,

    max_cps INTEGER NOT NULL DEFAULT 10,
    max_concurrent_calls INTEGER NOT NULL DEFAULT 100,
    max_daily_minutes BIGINT,

    codecs TEXT[] NOT NULL DEFAULT '{PCMU,PCMA}',
    supports_video BOOLEAN NOT NULL DEFAULT false,
    supports_fax BOOLEAN NOT NULL DEFAULT false,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_carrier_connections_org_name UNIQUE (organization_id, name),
    CONSTRAINT uq_carrier_connections_id_org UNIQUE (id, organization_id),
    CONSTRAINT uq_carrier_connections_id_provider UNIQUE (id, provider_id),
    CONSTRAINT chk_carrier_connections_scope CHECK (
        scope IN ('organization', 'platform')
    ),
    CONSTRAINT chk_carrier_connections_owner CHECK (
        (scope = 'organization' AND organization_id IS NOT NULL)
        OR (scope = 'platform' AND organization_id IS NULL)
    ),
    CONSTRAINT chk_carrier_connections_name CHECK (
        length(btrim(name)) > 0
    ),
    CONSTRAINT chk_carrier_connections_status CHECK (
        status IN ('active', 'disabled')
    ),
    CONSTRAINT chk_carrier_connections_auth_method CHECK (
        outbound_auth_method IN ('none', 'digest')
    ),
    CONSTRAINT chk_carrier_connections_auth_fields CHECK (
        (
            outbound_auth_method = 'none'
            AND auth_username IS NULL
            AND auth_secret_ciphertext IS NULL
        ) OR (
            outbound_auth_method = 'digest'
            AND auth_username IS NOT NULL
            AND length(btrim(auth_username)) > 0
            AND auth_secret_ciphertext IS NOT NULL
            AND length(auth_secret_ciphertext) > 0
        )
    ),
    CONSTRAINT chk_carrier_connections_cps CHECK (
        max_cps > 0
    ),
    CONSTRAINT chk_carrier_connections_concurrent CHECK (
        max_concurrent_calls > 0
    ),
    CONSTRAINT chk_carrier_connections_daily CHECK (
        max_daily_minutes IS NULL OR max_daily_minutes > 0
    ),
    CONSTRAINT chk_carrier_connections_inbound CHECK (
        inbound_auth_method IN ('ip', 'digest', 'none')
    ),
    CONSTRAINT chk_carrier_connections_inbound_auth_fields CHECK (
        (
            inbound_auth_method IN ('ip', 'none')
            AND inbound_username IS NULL
            AND inbound_secret_ciphertext IS NULL
        ) OR (
            inbound_auth_method = 'digest'
            AND inbound_username IS NOT NULL
            AND length(btrim(inbound_username)) > 0
            AND inbound_secret_ciphertext IS NOT NULL
            AND length(inbound_secret_ciphertext) > 0
        )
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_carrier_connections_platform_name
    ON carrier_connections (name)
    WHERE scope = 'platform';

CREATE INDEX IF NOT EXISTS idx_carrier_connections_organization_status
    ON carrier_connections (organization_id, status);

CREATE INDEX IF NOT EXISTS idx_carrier_connections_scope_status
    ON carrier_connections (scope, status);

CREATE INDEX IF NOT EXISTS idx_carrier_connections_provider_id
    ON carrier_connections (provider_id);

-- Provider-side resources that represent a platform carrier connection stay
-- internal. For DIDWW managed ingress, voice_in_trunk stores the DIDWW Voice IN
-- trunk assigned to the corresponding Leamout platform carrier connection.
CREATE TABLE IF NOT EXISTS carrier_connection_provider_resources (
    carrier_connection_id UUID NOT NULL,
    provider_id UUID NOT NULL,
    resource_type TEXT NOT NULL,
    provider_resource_id TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (carrier_connection_id, resource_type),
    CONSTRAINT fk_carrier_connection_provider_resources_connection
        FOREIGN KEY (carrier_connection_id, provider_id)
        REFERENCES carrier_connections (id, provider_id)
        ON DELETE CASCADE,
    CONSTRAINT uq_carrier_connection_provider_resources_provider_resource
        UNIQUE (provider_id, resource_type, provider_resource_id),
    CONSTRAINT chk_carrier_connection_provider_resources_type CHECK (
        resource_type IN ('voice_in_trunk')
    ),
    CONSTRAINT chk_carrier_connection_provider_resources_id CHECK (
        length(btrim(provider_resource_id)) > 0
    )
);

CREATE INDEX IF NOT EXISTS idx_carrier_connection_provider_resources_provider
    ON carrier_connection_provider_resources (provider_id, resource_type);

CREATE TRIGGER set_carrier_connection_provider_resources_updated_at
BEFORE UPDATE ON carrier_connection_provider_resources
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS carrier_connection_source_ips (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID,
    carrier_connection_id UUID NOT NULL REFERENCES carrier_connections(id) ON DELETE CASCADE,
    cidr CIDR NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_carrier_connection_source_ips_connection_cidr
        UNIQUE (carrier_connection_id, cidr)
);

CREATE INDEX IF NOT EXISTS idx_carrier_connection_source_ips_organization_id
    ON carrier_connection_source_ips (organization_id);

CREATE INDEX IF NOT EXISTS idx_carrier_connection_source_ips_cidr
    ON carrier_connection_source_ips USING gist (cidr inet_ops);

CREATE FUNCTION derive_carrier_source_ip_organization_id()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
BEGIN
    SELECT organization_id
    INTO NEW.organization_id
    FROM carrier_connections
    WHERE id = NEW.carrier_connection_id;
    RETURN NEW;
END;
$$;

CREATE TRIGGER derive_carrier_source_ip_organization_id
BEFORE INSERT OR UPDATE OF carrier_connection_id ON carrier_connection_source_ips
FOR EACH ROW
EXECUTE FUNCTION derive_carrier_source_ip_organization_id();

CREATE TRIGGER set_carrier_connections_updated_at
BEFORE UPDATE ON carrier_connections
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

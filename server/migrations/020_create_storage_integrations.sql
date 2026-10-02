CREATE TABLE IF NOT EXISTS storage_integrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    provider TEXT NOT NULL DEFAULT 's3',
    purpose TEXT NOT NULL DEFAULT 'recordings',
    endpoint_url TEXT NOT NULL,
    region TEXT NOT NULL DEFAULT 'us-east-1',
    bucket TEXT NOT NULL,
    access_key_id TEXT NOT NULL,
    secret_access_key_ciphertext TEXT NOT NULL,
    use_path_style BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'active',

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_storage_integrations_id_organization UNIQUE (id, organization_id),
    CONSTRAINT chk_storage_integrations_name CHECK (
        length(btrim(name)) BETWEEN 1 AND 128
    ),
    CONSTRAINT chk_storage_integrations_provider CHECK (
        provider IN ('s3')
    ),
    CONSTRAINT chk_storage_integrations_purpose CHECK (
        purpose IN ('recordings')
    ),
    CONSTRAINT chk_storage_integrations_endpoint CHECK (
        endpoint_url ~ '^https://[^[:space:]]+$'
    ),
    CONSTRAINT chk_storage_integrations_region CHECK (
        length(btrim(region)) BETWEEN 1 AND 128
    ),
    CONSTRAINT chk_storage_integrations_bucket CHECK (
        length(btrim(bucket)) BETWEEN 1 AND 255
    ),
    CONSTRAINT chk_storage_integrations_access_key CHECK (
        length(btrim(access_key_id)) > 0
    ),
    CONSTRAINT chk_storage_integrations_secret CHECK (
        length(btrim(secret_access_key_ciphertext)) > 0
    ),
    CONSTRAINT chk_storage_integrations_status CHECK (
        status IN ('active', 'disabled')
    )
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_storage_integrations_active_purpose
    ON storage_integrations (organization_id, purpose)
    WHERE status = 'active';

CREATE INDEX IF NOT EXISTS idx_storage_integrations_organization
    ON storage_integrations (organization_id, purpose, status);

CREATE TRIGGER set_storage_integrations_updated_at
BEFORE UPDATE ON storage_integrations
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

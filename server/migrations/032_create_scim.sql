CREATE TABLE IF NOT EXISTS scim_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    name TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    token_hash TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    last_used_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_scim_tokens_name UNIQUE (organization_id, name),
    CONSTRAINT uq_scim_tokens_hash UNIQUE (token_hash),
    CONSTRAINT chk_scim_tokens_name
        CHECK (length(btrim(name)) BETWEEN 1 AND 128),
    CONSTRAINT chk_scim_tokens_prefix
        CHECK (length(btrim(token_prefix)) BETWEEN 1 AND 32),
    CONSTRAINT chk_scim_tokens_hash
        CHECK (length(btrim(token_hash)) > 0)
);

CREATE INDEX IF NOT EXISTS idx_scim_tokens_organization
    ON scim_tokens (organization_id, created_at DESC);

CREATE TRIGGER set_scim_tokens_updated_at
BEFORE UPDATE ON scim_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS scim_identities (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    external_id TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_scim_identities_external
        UNIQUE (organization_id, external_id),
    CONSTRAINT uq_scim_identities_user
        UNIQUE (organization_id, user_id),
    CONSTRAINT fk_scim_identities_membership
        FOREIGN KEY (organization_id, user_id)
        REFERENCES organization_members(organization_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT chk_scim_identities_external_id
        CHECK (length(btrim(external_id)) BETWEEN 1 AND 255)
);

CREATE INDEX IF NOT EXISTS idx_scim_identities_user
    ON scim_identities (organization_id, user_id);

CREATE TRIGGER set_scim_identities_updated_at
BEFORE UPDATE ON scim_identities
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS organization_tokens (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    created_by UUID REFERENCES users(id) ON DELETE SET NULL,
    name TEXT NOT NULL,
    description TEXT,
    token_hash TEXT NOT NULL,
    token_prefix TEXT NOT NULL,
    scopes JSONB NOT NULL,
    last_used_at TIMESTAMPTZ,
    last_used_ip INET,
    expires_at TIMESTAMPTZ,
    disabled_at TIMESTAMPTZ,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_organization_tokens_scopes_is_array
        CHECK (jsonb_typeof(scopes) = 'array'),
    CONSTRAINT chk_organization_tokens_expires_after_created
        CHECK (expires_at IS NULL OR expires_at > created_at)
);

-- One active token per organization with the same name.
CREATE UNIQUE INDEX idx_organization_tokens_organization_name_active
    ON organization_tokens (organization_id, name)
    WHERE disabled_at IS NULL;

CREATE INDEX idx_organization_tokens_organization_id
    ON organization_tokens (organization_id);

CREATE UNIQUE INDEX idx_organization_tokens_token_hash
    ON organization_tokens (token_hash);

CREATE INDEX idx_organization_tokens_token_prefix
    ON organization_tokens (token_prefix);

CREATE TRIGGER set_organization_tokens_updated_at
BEFORE UPDATE ON organization_tokens
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

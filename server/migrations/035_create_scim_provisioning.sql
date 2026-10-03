CREATE TABLE IF NOT EXISTS scim_user_profiles (
    identity_id UUID PRIMARY KEY
        REFERENCES scim_identities(id)
        ON DELETE CASCADE,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL,

    user_name CITEXT NOT NULL,
    display_name TEXT,
    active BOOLEAN NOT NULL DEFAULT TRUE,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_scim_user_profiles_name
        UNIQUE (organization_id, user_name),
    CONSTRAINT uq_scim_user_profiles_identity_scope
        UNIQUE (identity_id, organization_id),
    CONSTRAINT uq_scim_user_profiles_user_scope
        UNIQUE (organization_id, user_id),
    CONSTRAINT fk_scim_user_profiles_membership
        FOREIGN KEY (organization_id, user_id)
        REFERENCES organization_members(organization_id, user_id)
        ON DELETE CASCADE,
    CONSTRAINT chk_scim_user_profiles_user_name
        CHECK (length(btrim(user_name::text)) BETWEEN 1 AND 320),
    CONSTRAINT chk_scim_user_profiles_display_name
        CHECK (display_name IS NULL OR length(btrim(display_name)) BETWEEN 1 AND 255)
);

CREATE INDEX IF NOT EXISTS idx_scim_user_profiles_organization
    ON scim_user_profiles (organization_id, created_at, identity_id);

CREATE TRIGGER set_scim_user_profiles_updated_at
BEFORE UPDATE ON scim_user_profiles
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS scim_groups (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    external_id TEXT,
    display_name TEXT NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_scim_groups_id_organization UNIQUE (id, organization_id),
    CONSTRAINT uq_scim_groups_name UNIQUE (organization_id, display_name),
    CONSTRAINT chk_scim_groups_external_id
        CHECK (external_id IS NULL OR length(btrim(external_id)) BETWEEN 1 AND 255),
    CONSTRAINT chk_scim_groups_display_name
        CHECK (length(btrim(display_name)) BETWEEN 1 AND 255)
);

CREATE UNIQUE INDEX IF NOT EXISTS uq_scim_groups_external
    ON scim_groups (organization_id, external_id)
    WHERE external_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_scim_groups_organization
    ON scim_groups (organization_id, created_at, id);

CREATE TRIGGER set_scim_groups_updated_at
BEFORE UPDATE ON scim_groups
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

CREATE TABLE IF NOT EXISTS scim_group_members (
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    group_id UUID NOT NULL,
    identity_id UUID NOT NULL,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    PRIMARY KEY (organization_id, group_id, identity_id),
    CONSTRAINT fk_scim_group_members_group_scope
        FOREIGN KEY (group_id, organization_id)
        REFERENCES scim_groups(id, organization_id)
        ON DELETE CASCADE,
    CONSTRAINT fk_scim_group_members_identity_scope
        FOREIGN KEY (identity_id, organization_id)
        REFERENCES scim_user_profiles(identity_id, organization_id)
        ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_scim_group_members_identity
    ON scim_group_members (organization_id, identity_id, group_id);

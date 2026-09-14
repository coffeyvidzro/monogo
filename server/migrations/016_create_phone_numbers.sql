CREATE TABLE IF NOT EXISTS phone_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,

    number TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    provisioning_mode TEXT NOT NULL DEFAULT 'byoc',
    carrier_connection_id UUID REFERENCES carrier_connections(id) ON DELETE SET NULL,
    provider_id UUID REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    provider_resource_id TEXT,
    voice_enabled BOOLEAN NOT NULL DEFAULT true,
    sms_enabled BOOLEAN NOT NULL DEFAULT false,
    status TEXT NOT NULL DEFAULT 'active',
    error_code TEXT,
    error_message TEXT,

    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT uq_phone_numbers_id_org_provider UNIQUE (
        id,
        organization_id,
        provider_id
    ),
    CONSTRAINT chk_phone_numbers_number CHECK (
        number ~ '^\+[1-9][0-9]{6,14}$'
    ),
    CONSTRAINT chk_phone_numbers_country_code CHECK (
        country_code ~ '^[A-Z]{2}$'
    ),
    CONSTRAINT chk_phone_numbers_provisioning_mode CHECK (
        provisioning_mode IN ('byoc', 'managed')
    ),
    CONSTRAINT chk_phone_numbers_status CHECK (
        status IN ('provisioning', 'active', 'disabled', 'porting', 'failed', 'released')
    ),
    CONSTRAINT chk_phone_numbers_provider_ownership CHECK (
        (
            provisioning_mode = 'byoc'
            AND provider_id IS NULL
            AND provider_resource_id IS NULL
            AND status NOT IN ('provisioning', 'failed')
        )
        OR (
            provisioning_mode = 'managed'
            AND provider_id IS NOT NULL
            AND (
                provider_resource_id IS NULL
                OR length(btrim(provider_resource_id)) > 0
            )
            AND (
                status IN ('provisioning', 'failed')
                OR provider_resource_id IS NOT NULL
            )
        )
    ),
    CONSTRAINT chk_phone_numbers_error_state CHECK (
        (
            status = 'failed'
            AND error_message IS NOT NULL
            AND length(btrim(error_message)) > 0
        )
        OR (
            status <> 'failed'
            AND error_code IS NULL
            AND error_message IS NULL
        )
    )
);

-- Failed/released rows are historical attempts or ownership. They must not
-- permanently reserve an E.164 number for a later acquisition.
CREATE UNIQUE INDEX IF NOT EXISTS uq_phone_numbers_number_live
    ON phone_numbers (number)
    WHERE status NOT IN ('failed', 'released');

CREATE UNIQUE INDEX IF NOT EXISTS uq_phone_numbers_provider_resource
    ON phone_numbers (provider_id, provider_resource_id)
    WHERE provider_id IS NOT NULL AND provider_resource_id IS NOT NULL;

CREATE INDEX IF NOT EXISTS idx_phone_numbers_organization_status
    ON phone_numbers (organization_id, status);

CREATE INDEX IF NOT EXISTS idx_phone_numbers_organization_provisioning_status
    ON phone_numbers (organization_id, provisioning_mode, status);

CREATE INDEX IF NOT EXISTS idx_phone_numbers_organization_country
    ON phone_numbers (organization_id, country_code);

CREATE INDEX IF NOT EXISTS idx_phone_numbers_number_prefix
    ON phone_numbers (number text_pattern_ops);

CREATE INDEX IF NOT EXISTS idx_phone_numbers_carrier_connection_id
    ON phone_numbers (carrier_connection_id)
    WHERE carrier_connection_id IS NOT NULL;

-- BYOC numbers may bind only to their organization's carrier connection.
-- Managed numbers may bind only to a Leamout platform connection owned by the
-- same upstream provider as the number.
CREATE FUNCTION validate_phone_number_carrier_scope()
RETURNS TRIGGER
LANGUAGE plpgsql
AS $$
DECLARE
    connection_scope TEXT;
    connection_organization_id UUID;
    connection_provider_id UUID;
BEGIN
    IF NEW.carrier_connection_id IS NULL THEN
        RETURN NEW;
    END IF;

    SELECT scope, organization_id, provider_id
    INTO connection_scope, connection_organization_id, connection_provider_id
    FROM carrier_connections
    WHERE id = NEW.carrier_connection_id;

    IF NEW.provisioning_mode = 'byoc' THEN
        IF connection_scope <> 'organization'
           OR connection_organization_id IS DISTINCT FROM NEW.organization_id THEN
            RAISE EXCEPTION 'BYOC phone number must use an organization-owned carrier connection'
                USING ERRCODE = '23514';
        END IF;
    ELSIF NEW.provisioning_mode = 'managed' THEN
        IF connection_scope <> 'platform'
           OR connection_provider_id IS DISTINCT FROM NEW.provider_id THEN
            RAISE EXCEPTION 'managed phone number must use a platform carrier connection for the same provider'
                USING ERRCODE = '23514';
        END IF;
    END IF;

    RETURN NEW;
END;
$$;

CREATE TRIGGER validate_phone_number_carrier_scope
BEFORE INSERT OR UPDATE OF carrier_connection_id, provisioning_mode, organization_id, provider_id
ON phone_numbers
FOR EACH ROW
EXECUTE FUNCTION validate_phone_number_carrier_scope();

CREATE TRIGGER set_phone_numbers_updated_at
BEFORE UPDATE ON phone_numbers
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

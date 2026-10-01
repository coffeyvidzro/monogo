CREATE TABLE IF NOT EXISTS phone_numbers (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    number TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    carrier_connection_id UUID,
    voice_enabled BOOLEAN NOT NULL DEFAULT true,
    status TEXT NOT NULL DEFAULT 'active',
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    CONSTRAINT uq_phone_numbers_id_organization UNIQUE (id, organization_id),
    CONSTRAINT fk_phone_numbers_carrier_connection_scope
        FOREIGN KEY (carrier_connection_id, organization_id)
        REFERENCES carrier_connections(id, organization_id)
        ON DELETE SET NULL,
    CONSTRAINT chk_phone_numbers_number CHECK (number ~ '^\+[1-9][0-9]{6,14}$'),
    CONSTRAINT chk_phone_numbers_country_code CHECK (country_code ~ '^[A-Z]{2}$'),
    CONSTRAINT chk_phone_numbers_status CHECK (status IN ('active', 'disabled', 'porting', 'released'))
);
CREATE UNIQUE INDEX IF NOT EXISTS uq_phone_numbers_number_live ON phone_numbers (number) WHERE status <> 'released';
CREATE INDEX IF NOT EXISTS idx_phone_numbers_organization_status ON phone_numbers (organization_id, status);
CREATE INDEX IF NOT EXISTS idx_phone_numbers_organization_country ON phone_numbers (organization_id, country_code);
CREATE INDEX IF NOT EXISTS idx_phone_numbers_number_prefix ON phone_numbers (number text_pattern_ops);
CREATE INDEX IF NOT EXISTS idx_phone_numbers_carrier_connection_id ON phone_numbers (carrier_connection_id) WHERE carrier_connection_id IS NOT NULL;

CREATE TRIGGER set_phone_numbers_updated_at BEFORE UPDATE ON phone_numbers FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Composite references below make tenant ownership a database invariant.
ALTER TABLE phone_numbers
    ADD CONSTRAINT uq_phone_numbers_id_organization UNIQUE (id, organization_id);

-- Long-running number changes are modeled separately from phone_numbers so a
-- provider callback can be correlated and every transition remains auditable.
CREATE TABLE number_lifecycle_operations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    phone_number_id UUID,
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    idempotency_key TEXT NOT NULL,
    operation TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_reference TEXT,
    requested_number TEXT NOT NULL,
    request_payload JSONB NOT NULL DEFAULT '{}'::JSONB,
    failure_code TEXT,
    failure_message TEXT,
    submitted_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (organization_id, idempotency_key),
    FOREIGN KEY (phone_number_id, organization_id)
        REFERENCES phone_numbers(id, organization_id) ON DELETE RESTRICT,
    CHECK (length(btrim(idempotency_key)) BETWEEN 1 AND 255),
    CHECK (requested_number ~ '^\+[1-9][0-9]{6,14}$'),
    CHECK (operation IN ('assign', 'port_in', 'port_out', 'release')),
    CHECK (status IN ('pending', 'submitted', 'in_progress', 'completed', 'failed', 'manual_review')),
    CHECK (status NOT IN ('submitted', 'in_progress', 'completed') OR submitted_at IS NOT NULL),
    CHECK (status <> 'completed' OR completed_at IS NOT NULL),
    CHECK ((status = 'failed' AND failure_message IS NOT NULL)
           OR (status <> 'failed' AND failure_code IS NULL AND failure_message IS NULL))
);

CREATE UNIQUE INDEX uq_number_lifecycle_provider_reference
    ON number_lifecycle_operations (provider_id, provider_reference)
    WHERE provider_reference IS NOT NULL;
CREATE INDEX idx_number_lifecycle_reconciliation
    ON number_lifecycle_operations (updated_at)
    WHERE status IN ('submitted', 'in_progress');

CREATE TRIGGER set_number_lifecycle_operations_updated_at
BEFORE UPDATE ON number_lifecycle_operations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

-- Emergency addresses are versioned instead of overwritten. Only one current
-- registration can exist for a number, while old rows provide an audit trail.
CREATE TABLE emergency_registrations (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    phone_number_id UUID NOT NULL,
    provider_id UUID NOT NULL REFERENCES carrier_providers(id) ON DELETE RESTRICT,
    status TEXT NOT NULL DEFAULT 'pending',
    name TEXT NOT NULL,
    address_line1 TEXT NOT NULL,
    address_line2 TEXT,
    locality TEXT NOT NULL,
    region TEXT NOT NULL,
    postal_code TEXT NOT NULL,
    country_code CHAR(2) NOT NULL,
    provider_reference TEXT,
    validation_message TEXT,
    activated_at TIMESTAMPTZ,
    deactivated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (phone_number_id, organization_id)
        REFERENCES phone_numbers(id, organization_id) ON DELETE RESTRICT,
    CHECK (status IN ('pending', 'validating', 'active', 'rejected', 'deactivated')),
    CHECK (country_code ~ '^[A-Z]{2}$'),
    CHECK (status <> 'active' OR (provider_reference IS NOT NULL AND activated_at IS NOT NULL)),
    CHECK (status <> 'deactivated' OR deactivated_at IS NOT NULL)
);

CREATE UNIQUE INDEX uq_emergency_registrations_current
    ON emergency_registrations (phone_number_id)
    WHERE status IN ('pending', 'validating', 'active');
CREATE UNIQUE INDEX uq_emergency_registrations_provider_reference
    ON emergency_registrations (provider_id, provider_reference)
    WHERE provider_reference IS NOT NULL;

CREATE TRIGGER set_emergency_registrations_updated_at
BEFORE UPDATE ON emergency_registrations
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

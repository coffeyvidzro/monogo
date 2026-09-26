ALTER TABLE number_lifecycle_operations
    ADD COLUMN reconcile_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN reconcile_attempts INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_number_lifecycle_reconcile_attempts CHECK (reconcile_attempts >= 0);

CREATE INDEX idx_number_lifecycle_due
    ON number_lifecycle_operations (reconcile_after, created_at)
    WHERE status IN ('pending', 'submitted', 'in_progress');

ALTER TABLE emergency_registrations
    ADD COLUMN idempotency_key TEXT,
    ADD COLUMN request_hash TEXT,
    ADD COLUMN reconcile_after TIMESTAMPTZ NOT NULL DEFAULT now(),
    ADD COLUMN reconcile_attempts INTEGER NOT NULL DEFAULT 0,
    ADD CONSTRAINT chk_emergency_registration_idempotency
        CHECK (idempotency_key IS NULL OR length(btrim(idempotency_key)) BETWEEN 1 AND 255),
    ADD CONSTRAINT chk_emergency_registration_hash
        CHECK (request_hash IS NULL OR request_hash ~ '^[0-9a-f]{64}$'),
    ADD CONSTRAINT chk_emergency_registration_reconcile_attempts
        CHECK (reconcile_attempts >= 0),
    ADD CONSTRAINT uq_emergency_registration_idempotency
        UNIQUE (organization_id, idempotency_key);

CREATE INDEX idx_emergency_registration_due
    ON emergency_registrations (reconcile_after, created_at)
    WHERE status IN ('pending', 'validating');

CREATE TABLE port_in_cases (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    lifecycle_operation_id UUID NOT NULL UNIQUE REFERENCES number_lifecycle_operations(id) ON DELETE RESTRICT,
    losing_carrier TEXT NOT NULL,
    account_number TEXT NOT NULL,
    account_pin_ciphertext BYTEA,
    authorized_name TEXT NOT NULL,
    service_address JSONB NOT NULL,
    desired_port_date DATE,
    status TEXT NOT NULL DEFAULT 'draft',
    provider_case_reference TEXT,
    rejection_code TEXT,
    rejection_message TEXT,
    foc_at TIMESTAMPTZ,
    activated_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    UNIQUE (id, organization_id),
    CHECK (length(btrim(losing_carrier)) > 0),
    CHECK (length(btrim(account_number)) > 0),
    CHECK (length(btrim(authorized_name)) > 0),
    CHECK (status IN ('draft', 'checking_portability', 'documents_required', 'submitted',
                      'in_progress', 'foc_received', 'activated', 'rejected', 'cancelled')),
    CHECK (status <> 'foc_received' OR foc_at IS NOT NULL),
    CHECK (status <> 'activated' OR activated_at IS NOT NULL),
    CHECK ((status = 'rejected' AND rejection_message IS NOT NULL)
           OR (status <> 'rejected' AND rejection_code IS NULL AND rejection_message IS NULL))
);

CREATE UNIQUE INDEX uq_port_in_provider_case
    ON port_in_cases (provider_case_reference)
    WHERE provider_case_reference IS NOT NULL;

CREATE TRIGGER set_port_in_cases_updated_at
BEFORE UPDATE ON port_in_cases
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

CREATE TABLE port_in_documents (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE RESTRICT,
    port_in_case_id UUID NOT NULL,
    document_type TEXT NOT NULL,
    object_key TEXT NOT NULL,
    sha256 TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'pending',
    provider_reference TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    FOREIGN KEY (port_in_case_id, organization_id)
        REFERENCES port_in_cases(id, organization_id) ON DELETE RESTRICT,
    UNIQUE (port_in_case_id, document_type, sha256),
    CHECK (document_type IN ('loa', 'invoice', 'ownership', 'identity', 'other')),
    CHECK (length(btrim(object_key)) > 0),
    CHECK (sha256 ~ '^[0-9a-f]{64}$'),
    CHECK (status IN ('pending', 'accepted', 'rejected'))
);

CREATE TRIGGER set_port_in_documents_updated_at
BEFORE UPDATE ON port_in_documents
FOR EACH ROW EXECUTE FUNCTION set_updated_at();

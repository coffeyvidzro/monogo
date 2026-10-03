ALTER TABLE ai_provider_credentials
    ADD COLUMN connection_state TEXT NOT NULL DEFAULT 'unchecked',
    ADD COLUMN verified_at TIMESTAMPTZ,
    ADD COLUMN failure_code TEXT,
    ADD CONSTRAINT chk_ai_provider_credentials_connection_state
        CHECK (connection_state IN ('unchecked', 'ready', 'invalid', 'unavailable')),
    ADD CONSTRAINT chk_ai_provider_credentials_failure_code
        CHECK (failure_code IS NULL OR length(btrim(failure_code)) BETWEEN 1 AND 64);

CREATE INDEX idx_ai_provider_credentials_organization_state
    ON ai_provider_credentials (organization_id, connection_state, provider);

CREATE TABLE idempotency (
    scope TEXT NOT NULL,
    idempotency_key TEXT NOT NULL,
    method TEXT NOT NULL,
    path TEXT NOT NULL,
    request_hash TEXT NOT NULL,
    status TEXT NOT NULL DEFAULT 'processing',
    response_status INTEGER,
    response_body BYTEA,
    response_content_type TEXT,
    response_headers JSONB NOT NULL DEFAULT '{}'::JSONB,
    locked_until TIMESTAMPTZ NOT NULL,
    completed_at TIMESTAMPTZ,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),

    CONSTRAINT idempotency_pkey PRIMARY KEY (scope, idempotency_key),
    CONSTRAINT chk_idempotency_scope CHECK (length(trim(scope)) BETWEEN 1 AND 255),
    CONSTRAINT chk_idempotency_key CHECK (length(trim(idempotency_key)) BETWEEN 1 AND 255),
    CONSTRAINT chk_idempotency_method CHECK (length(trim(method)) > 0),
    CONSTRAINT chk_idempotency_path CHECK (length(trim(path)) > 0),
    CONSTRAINT chk_idempotency_request_hash CHECK (length(request_hash) = 64),
    CONSTRAINT chk_idempotency_status CHECK (status IN ('processing', 'completed')),
    CONSTRAINT chk_idempotency_response_headers CHECK (jsonb_typeof(response_headers) = 'object'),
    CONSTRAINT chk_idempotency_response_status CHECK (
        response_status IS NULL OR response_status BETWEEN 100 AND 599
    ),
    CONSTRAINT chk_idempotency_completion CHECK (
        (status = 'processing' AND response_status IS NULL AND completed_at IS NULL)
        OR
        (status = 'completed' AND response_status IS NOT NULL AND completed_at IS NOT NULL)
    ),
    CONSTRAINT chk_idempotency_expiration CHECK (
        expires_at > created_at AND locked_until <= expires_at
    )
);

CREATE INDEX idx_idempotency_expires_at ON idempotency (expires_at);

COMMENT ON TABLE idempotency IS
    'Durable request replay records scoped to an authenticated principal or organization.';

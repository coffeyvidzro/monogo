CREATE TABLE IF NOT EXISTS recordings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    call_id UUID NOT NULL REFERENCES calls(id) ON DELETE CASCADE,
    status TEXT NOT NULL DEFAULT 'recording',
    storage_key TEXT,
    storage_provider TEXT,
    storage_bucket TEXT,
    storage_url TEXT,
    file_size_bytes BIGINT,
    format TEXT,
    duration_seconds INTEGER,
    started_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),

    CONSTRAINT chk_recordings_status CHECK (
        status IN ('recording', 'completed', 'failed', 'deleted')
    ),
    CONSTRAINT chk_recordings_duration CHECK (
        duration_seconds IS NULL OR duration_seconds >= 0
    ),
    CONSTRAINT chk_recordings_file_size CHECK (
        file_size_bytes IS NULL OR file_size_bytes >= 0
    ),
    CONSTRAINT chk_recordings_completed_at CHECK (
        completed_at IS NULL OR started_at IS NULL OR completed_at >= started_at
    )
);

CREATE INDEX IF NOT EXISTS idx_recordings_organization_created
    ON recordings (organization_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_recordings_call_created
    ON recordings (call_id, created_at DESC);

CREATE INDEX IF NOT EXISTS idx_recordings_status
    ON recordings (organization_id, status, created_at DESC);

CREATE UNIQUE INDEX IF NOT EXISTS uq_recordings_call_storage_key
    ON recordings (call_id, storage_key)
    WHERE storage_key IS NOT NULL;

CREATE TRIGGER set_recordings_updated_at
BEFORE UPDATE ON recordings
FOR EACH ROW
EXECUTE FUNCTION set_updated_at();

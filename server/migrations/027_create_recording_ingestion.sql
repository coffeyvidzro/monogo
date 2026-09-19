ALTER TABLE recordings
    ADD COLUMN source_path TEXT,
    ADD COLUMN stopped_at TIMESTAMPTZ,
    ADD COLUMN upload_attempts INTEGER NOT NULL DEFAULT 0,
    ADD COLUMN next_upload_at TIMESTAMPTZ,
    ADD COLUMN upload_error TEXT;

UPDATE recordings
SET source_path = storage_key,
    storage_key = NULL,
    storage_provider = NULL
WHERE storage_provider = 'freeswitch-local';

ALTER TABLE recordings
    DROP CONSTRAINT chk_recordings_status,
    ADD CONSTRAINT chk_recordings_status CHECK (
        status IN ('recording', 'uploading', 'completed', 'failed', 'deleted')
    ),
    ADD CONSTRAINT chk_recordings_upload_attempts CHECK (upload_attempts >= 0),
    ADD CONSTRAINT chk_recordings_source_path CHECK (
        source_path IS NULL OR (source_path LIKE '/%' AND source_path !~ '(^|/)\.\.(/|$)')
    );

CREATE INDEX idx_recordings_upload_queue
    ON recordings (next_upload_at, updated_at)
    WHERE status = 'uploading';

CREATE UNIQUE INDEX uq_recordings_call_source_path
    ON recordings (call_id, source_path)
    WHERE source_path IS NOT NULL;

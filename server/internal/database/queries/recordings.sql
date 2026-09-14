-- name: CreateRecording :one
INSERT INTO recordings (
    organization_id,
    call_id,
    status,
    storage_key,
    storage_provider,
    storage_bucket,
    storage_url,
    file_size_bytes,
    format,
    started_at
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(call_id),
    COALESCE(sqlc.narg(status), 'recording'),
    sqlc.narg(storage_key),
    sqlc.narg(storage_provider),
    sqlc.narg(storage_bucket),
    sqlc.narg(storage_url),
    sqlc.narg(file_size_bytes),
    sqlc.narg(format),
    COALESCE(sqlc.narg(started_at), NOW())
)
RETURNING *;

-- name: GetRecording :one
SELECT *
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status <> 'deleted'
LIMIT 1;

-- name: GetRecordingIncludingDeleted :one
SELECT *
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetRecordingByCallStorageKey :one
SELECT *
FROM recordings
WHERE call_id = sqlc.arg(call_id)
  AND storage_key = sqlc.arg(storage_key)
LIMIT 1;

-- name: ListCallRecordings :many
SELECT *
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND call_id = sqlc.arg(call_id)
  AND status <> 'deleted'
ORDER BY created_at DESC;

-- name: ListRecordingsForReconciliation :many
SELECT r.*
FROM recordings r
JOIN calls c ON c.id = r.call_id
WHERE r.status = 'recording'
  AND c.state IN ('completed', 'failed', 'cancelled')
  AND r.updated_at <= sqlc.arg(updated_before)
ORDER BY r.updated_at ASC
LIMIT sqlc.arg(batch_size);

-- name: CompleteRecording :one
UPDATE recordings
SET
    status = 'completed',
    storage_key = COALESCE(sqlc.narg(storage_key), storage_key),
    storage_provider = COALESCE(sqlc.narg(storage_provider), storage_provider),
    storage_bucket = COALESCE(sqlc.narg(storage_bucket), storage_bucket),
    storage_url = COALESCE(sqlc.narg(storage_url), storage_url),
    file_size_bytes = COALESCE(sqlc.narg(file_size_bytes), file_size_bytes),
    format = COALESCE(sqlc.narg(format), format),
    duration_seconds = COALESCE(sqlc.narg(duration_seconds), duration_seconds),
    completed_at = COALESCE(completed_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'recording'
RETURNING *;

-- name: FailRecording :one
UPDATE recordings
SET
    status = 'failed',
    completed_at = COALESCE(completed_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status = 'recording'
RETURNING *;

-- name: ListRecordings :many
SELECT *
FROM recordings
WHERE organization_id = sqlc.arg(organization_id)
  AND status <> 'deleted'
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: DeleteRecording :one
UPDATE recordings
SET
    status = 'deleted',
    storage_url = NULL,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND status <> 'deleted'
RETURNING *;
-- name: ListBackofficeRecordings :many
SELECT r.id::TEXT AS id, r.organization_id::TEXT AS organization_id, o.name AS organization_name,
       r.call_id::TEXT AS call_id, r.status, COALESCE(r.format,'—') AS format,
       CAST(COALESCE(r.duration_seconds::TEXT,'—') AS TEXT) AS duration_seconds,
       CAST(COALESCE(r.file_size_bytes::TEXT,'—') AS TEXT) AS file_size_bytes,
       to_char(r.created_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI') AS created_at
FROM recordings r JOIN organizations o ON o.id=r.organization_id
ORDER BY r.created_at DESC LIMIT 100;

-- name: GetBackofficeRecording :one
SELECT r.id::TEXT AS id, r.organization_id::TEXT AS organization_id, o.name AS organization_name,
       r.call_id::TEXT AS call_id, r.status, COALESCE(r.storage_provider,'—') AS storage_provider,
       COALESCE(r.storage_bucket,'—') AS storage_bucket, COALESCE(r.storage_key,'—') AS storage_key,
       COALESCE(r.storage_url,'—') AS storage_url, COALESCE(r.format,'—') AS format,
       CAST(COALESCE(r.duration_seconds::TEXT,'—') AS TEXT) AS duration_seconds,
       CAST(COALESCE(r.file_size_bytes::TEXT,'—') AS TEXT) AS file_size_bytes,
       CAST(COALESCE(to_char(r.started_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS started_at,
       CAST(COALESCE(to_char(r.completed_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS completed_at,
       to_char(r.created_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI') AS created_at,
       to_char(r.updated_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI') AS updated_at
FROM recordings r JOIN organizations o ON o.id=r.organization_id WHERE r.id=sqlc.arg(id) LIMIT 1;

# Recording storage operations

## Architecture

FreeSWITCH writes recordings to the shared staging volume. The worker receives
`RECORD_STOP`, marks the database row `uploading`, and uploads the closed file to
the private `recordings` MinIO bucket. Only after the object upload and database
transaction succeed does the worker remove the staged file.

Objects use tenant-partitioned, opaque keys:

```text
recordings/{organization_id}/{yyyy}/{mm}/{dd}/{recording_id}.{format}
```

The API never exposes the internal MinIO endpoint or permanent object URLs.
Playback creates a short-lived, signed URL for `https://recordings.${DOMAIN}`.

## Deployment

Set unique values for all four MinIO credentials in the deployment environment:

```text
MINIO_ROOT_USER=...
MINIO_ROOT_PASSWORD=...
MINIO_APP_ACCESS_KEY=...
MINIO_APP_SECRET_KEY=...
```

The one-shot `minio-init` service creates a private bucket and a bucket-scoped
application policy. The API and worker use the application credentials, not the
root credentials. MinIO's console is not published. Caddy exposes only the S3
API hostname needed by signed playback URLs.

The FreeSWITCH and worker containers share `recordings-data`. Do not mount this
volume into the API container. The worker validates that every source file and
resolved symlink remains beneath `S3_STAGING_PATH` before opening it.

## Retry and recovery

Upload work is leased in batches so multiple workers do not normally process the
same row. Failed uploads use exponential backoff from 5 seconds to 15 minutes.
After 10 attempts, the recording is marked `failed`. A successful retry uploads
to the same deterministic key, so an interrupted database commit is safe to
retry.

The recording reconciliation job moves stale `recording` rows to `uploading`
after their call reaches a terminal state. This recovers from a missing
FreeSWITCH `RECORD_STOP` event. Operators should alert on:

- `uploading` rows whose `next_upload_at` remains in the past;
- rising `upload_attempts` or non-empty `upload_error`;
- `failed` recordings;
- growth of the staging volume;
- MinIO capacity, healing, or drive errors.

## Playback and deletion

Playback is available only for `completed` recordings whose provider, bucket,
and key match the configured S3 store. Signed URLs expire after 15 minutes by
default.

Deletion removes the object before marking the row deleted. If object deletion
fails, the row remains visible and the API returns a service-unavailable error,
allowing the caller to retry without losing the object reference.

## Backup and restore

The named MinIO volume is durable across container restarts but is not a backup.
Production deployments must replicate or back up both PostgreSQL and MinIO and
test a coordinated restore. Restoring only one side can leave database rows
without objects or unreferenced objects in the bucket.

-- name: CreateStorageIntegration :one
INSERT INTO storage_integrations (
    id,
    organization_id,
    name,
    provider,
    purpose,
    endpoint_url,
    region,
    bucket,
    access_key_id,
    secret_access_key_ciphertext,
    use_path_style,
    status
)
SELECT
    sqlc.arg(id),
    sqlc.arg(organization_id),
    sqlc.arg(name),
    's3',
    'recordings',
    sqlc.arg(endpoint_url),
    sqlc.arg(region),
    sqlc.arg(bucket),
    sqlc.arg(access_key_id),
    sqlc.arg(secret_access_key_ciphertext),
    sqlc.arg(use_path_style),
    'active'
FROM organizations AS o
WHERE o.id = sqlc.arg(organization_id)
  AND o.status = 'active'
  AND o.deleted_at IS NULL
RETURNING *;

-- name: ListStorageIntegrations :many
SELECT *
FROM storage_integrations
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC;

-- name: GetStorageIntegration :one
SELECT *
FROM storage_integrations
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
LIMIT 1;

-- name: GetActiveRecordingStorageIntegration :one
SELECT *
FROM storage_integrations
WHERE organization_id = sqlc.arg(organization_id)
  AND purpose = 'recordings'
  AND status = 'active'
LIMIT 1;

-- name: UpdateStorageIntegration :one
UPDATE storage_integrations
SET
    name = COALESCE(sqlc.narg(name), name),
    endpoint_url = COALESCE(sqlc.narg(endpoint_url), endpoint_url),
    region = COALESCE(sqlc.narg(region), region),
    bucket = COALESCE(sqlc.narg(bucket), bucket),
    access_key_id = COALESCE(sqlc.narg(access_key_id), access_key_id),
    secret_access_key_ciphertext = COALESCE(
        sqlc.narg(secret_access_key_ciphertext),
        secret_access_key_ciphertext
    ),
    use_path_style = COALESCE(sqlc.narg(use_path_style), use_path_style),
    status = COALESCE(sqlc.narg(status), status),
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

-- name: DisableStorageIntegration :one
UPDATE storage_integrations
SET status = 'disabled',
    updated_at = NOW()
WHERE id = sqlc.arg(id)
  AND organization_id = sqlc.arg(organization_id)
RETURNING *;

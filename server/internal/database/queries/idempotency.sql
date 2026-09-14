-- name: ClaimIdempotencyKey :one
INSERT INTO idempotency (
    scope, idempotency_key, method, path, request_hash,
    locked_until, expires_at
)
VALUES (
    sqlc.arg(scope), sqlc.arg(idempotency_key), sqlc.arg(method),
    sqlc.arg(path), sqlc.arg(request_hash), sqlc.arg(locked_until),
    sqlc.arg(expires_at)
)
ON CONFLICT (scope, idempotency_key) DO UPDATE
SET
    method = EXCLUDED.method,
    path = EXCLUDED.path,
    request_hash = EXCLUDED.request_hash,
    status = 'processing',
    response_status = NULL,
    response_body = NULL,
    response_content_type = NULL,
    response_headers = '{}'::JSONB,
    locked_until = EXCLUDED.locked_until,
    completed_at = NULL,
    expires_at = EXCLUDED.expires_at,
    updated_at = NOW()
WHERE idempotency.expires_at <= NOW()
   OR (
       idempotency.status = 'processing'
       AND idempotency.locked_until <= NOW()
       AND idempotency.method = EXCLUDED.method
       AND idempotency.path = EXCLUDED.path
       AND idempotency.request_hash = EXCLUDED.request_hash
   )
RETURNING *;

-- name: GetIdempotencyKey :one
SELECT *
FROM idempotency
WHERE scope = sqlc.arg(scope)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: CompleteIdempotencyKey :one
UPDATE idempotency
SET
    status = 'completed',
    response_status = sqlc.arg(response_status),
    response_body = sqlc.arg(response_body),
    response_content_type = sqlc.narg(response_content_type),
    response_headers = sqlc.arg(response_headers),
    completed_at = NOW(),
    locked_until = NOW(),
    updated_at = NOW()
WHERE scope = sqlc.arg(scope)
  AND idempotency_key = sqlc.arg(idempotency_key)
  AND method = sqlc.arg(method)
  AND path = sqlc.arg(path)
  AND request_hash = sqlc.arg(request_hash)
  AND status = 'processing'
  AND locked_until = sqlc.arg(locked_until)
RETURNING *;

-- name: DeleteExpiredIdempotencyKeys :execrows
DELETE FROM idempotency
WHERE expires_at <= NOW();

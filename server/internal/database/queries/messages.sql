-- name: CreateOutboundMessage :one
INSERT INTO messages (
    organization_id,
    channel,
    direction,
    status,
    from_address,
    to_address,
    body,
    media,
    idempotency_key,
    request_hash,
    queued_at
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(channel),
    'outbound',
    'queued',
    sqlc.arg(from_address),
    sqlc.arg(to_address),
    sqlc.narg(body),
    sqlc.arg(media)::jsonb,
    sqlc.arg(idempotency_key),
    sqlc.arg(request_hash),
    NOW()
)
ON CONFLICT (organization_id, idempotency_key)
WHERE idempotency_key IS NOT NULL
DO NOTHING
RETURNING *;

-- name: CreateInboundMessage :one
INSERT INTO messages (
    organization_id,
    messaging_connection_id,
    channel,
    direction,
    status,
    from_address,
    to_address,
    body,
    media,
    provider_message_id,
    received_at
)
SELECT
    sqlc.arg(organization_id),
    connection.id,
    sqlc.arg(channel),
    'inbound',
    'received',
    sqlc.arg(from_address),
    sqlc.arg(to_address),
    sqlc.narg(body),
    sqlc.arg(media)::jsonb,
    sqlc.arg(provider_message_id),
    NOW()
FROM messaging_connections AS connection
JOIN organizations AS organization
  ON organization.id = sqlc.arg(organization_id)
WHERE connection.id = sqlc.arg(messaging_connection_id)
  AND connection.status = 'active'
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
  AND (
    (connection.scope = 'organization' AND connection.organization_id = sqlc.arg(organization_id))
    OR (connection.scope = 'platform' AND connection.organization_id IS NULL)
  )
ON CONFLICT (messaging_connection_id, provider_message_id)
WHERE messaging_connection_id IS NOT NULL
  AND provider_message_id IS NOT NULL
DO NOTHING
RETURNING *;

-- name: GetMessage :one
SELECT *
FROM messages
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetMessageByIdempotencyKey :one
SELECT *
FROM messages
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key)
LIMIT 1;

-- name: GetMessageByProviderID :one
SELECT *
FROM messages
WHERE messaging_connection_id = sqlc.arg(messaging_connection_id)
  AND provider_message_id = sqlc.arg(provider_message_id)
LIMIT 1;

-- name: ListMessages :many
SELECT *
FROM messages
WHERE organization_id = sqlc.arg(organization_id)
  AND (sqlc.narg(status)::text IS NULL OR status = sqlc.narg(status)::text)
  AND (sqlc.narg(direction)::text IS NULL OR direction = sqlc.narg(direction)::text)
  AND (sqlc.narg(channel)::text IS NULL OR channel = sqlc.narg(channel)::text)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: SetMessageProviderAttribution :one
UPDATE messages AS message
SET
    messaging_connection_id = connection.id,
    updated_at = NOW()
FROM messaging_connections AS connection
JOIN organizations AS organization
  ON organization.id = sqlc.arg(organization_id)
WHERE message.organization_id = sqlc.arg(organization_id)
  AND message.id = sqlc.arg(id)
  AND message.direction = 'outbound'
  AND message.status = 'queued'
  AND connection.id = sqlc.arg(messaging_connection_id)
  AND connection.status = 'active'
  AND organization.status = 'active'
  AND organization.deleted_at IS NULL
  AND (
    (connection.scope = 'organization' AND connection.organization_id = message.organization_id)
    OR (connection.scope = 'platform' AND connection.organization_id IS NULL)
  )
  AND (
      message.messaging_connection_id IS NULL
      OR message.messaging_connection_id = connection.id
  )
RETURNING message.*;

-- name: BeginMessageSubmission :one
UPDATE messages
SET status = 'submitting', submitting_at = NOW(), updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status = 'queued'
  AND messaging_connection_id IS NOT NULL
RETURNING *;

-- name: RequeueMessageSubmission :one
UPDATE messages
SET
    status = 'queued',
    submitting_at = NULL,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status = 'submitting'
RETURNING *;

-- name: MarkMessageSubmitted :one
UPDATE messages
SET
    status = 'submitted',
    provider_message_id = sqlc.arg(provider_message_id),
    submitted_at = COALESCE(submitted_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status = 'submitting'
  AND messaging_connection_id IS NOT NULL
  AND provider_message_id IS NULL
RETURNING *;

-- name: MarkMessageSent :one
UPDATE messages
SET
    status = 'sent',
    sent_at = COALESCE(sent_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status = 'submitted'
RETURNING *;

-- name: MarkMessageDelivered :one
UPDATE messages
SET
    status = 'delivered',
    delivered_at = COALESCE(delivered_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status IN ('submitted', 'sent')
RETURNING *;

-- name: MarkMessageUndelivered :one
UPDATE messages
SET
    status = 'undelivered',
    failure_code = sqlc.narg(failure_code),
    failure_message = sqlc.narg(failure_message),
    failed_at = COALESCE(failed_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status IN ('submitted', 'sent')
RETURNING *;

-- name: MarkMessageFailed :one
UPDATE messages
SET
    status = 'failed',
    failure_code = sqlc.narg(failure_code),
    failure_message = sqlc.narg(failure_message),
    failed_at = COALESCE(failed_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status IN ('queued', 'submitting', 'submitted', 'sent')
RETURNING *;


-- name: MarkMessageSubmissionUnknown :one
UPDATE messages
SET
    status = 'submission_unknown',
    submission_unknown_at = NOW(),
    failure_code = sqlc.narg(failure_code),
    failure_message = sqlc.narg(failure_message),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND direction = 'outbound'
  AND status = 'submitting'
RETURNING *;

-- name: ListStaleMessageSubmissions :many
SELECT * FROM messages
WHERE status = 'submitting' AND submitting_at < sqlc.arg(stale_before)
ORDER BY submitting_at ASC;

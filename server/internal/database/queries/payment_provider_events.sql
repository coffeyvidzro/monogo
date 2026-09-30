-- name: CreatePaymentProviderEvent :one
INSERT INTO payment_provider_events (
    payment_id,
    organization_id,
    provider,
    provider_event_id,
    event_type,
    payload_sha256,
    payload,
    received_at
)
VALUES (
    sqlc.arg(payment_id),
    sqlc.arg(organization_id),
    sqlc.arg(provider),
    sqlc.arg(provider_event_id),
    sqlc.arg(event_type),
    sqlc.arg(payload_sha256),
    sqlc.arg(payload),
    sqlc.arg(received_at)
)
RETURNING *;

-- name: GetPaymentProviderEventByIdentity :one
SELECT *
FROM payment_provider_events
WHERE provider = sqlc.arg(provider)
  AND provider_event_id = sqlc.arg(provider_event_id)
LIMIT 1;

-- name: MarkPaymentProviderEventProcessed :one
WITH updated AS (
    UPDATE payment_provider_events AS event
    SET processed_at = sqlc.arg(processed_at)
    WHERE event.id = sqlc.arg(id)
      AND event.processed_at IS NULL
    RETURNING event.*
)
SELECT * FROM updated
UNION ALL
SELECT event.*
FROM payment_provider_events AS event
WHERE event.id = sqlc.arg(id)
  AND event.processed_at IS NOT NULL
LIMIT 1;

-- name: ListUnprocessedPaymentProviderEvents :many
SELECT *
FROM payment_provider_events
WHERE processed_at IS NULL
ORDER BY received_at, id
LIMIT sqlc.arg(limit_count);

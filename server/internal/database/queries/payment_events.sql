-- name: CreatePaymentEvent :one
INSERT INTO payment_events (
    payment_id,
    organization_id,
    provider,
    provider_event_id,
    event_type,
    failure_code,
    payload,
    occurred_at
)
VALUES (
    sqlc.arg(payment_id),
    sqlc.arg(organization_id),
    sqlc.arg(provider),
    sqlc.arg(provider_event_id),
    sqlc.arg(event_type),
    sqlc.narg(failure_code),
    sqlc.arg(payload),
    sqlc.arg(occurred_at)
)
RETURNING *;

-- name: GetPaymentEventByProviderEventID :one
SELECT *
FROM payment_events
WHERE provider = sqlc.arg(provider)
  AND provider_event_id = sqlc.arg(provider_event_id)
LIMIT 1;

-- name: ListPaymentEventsByPayment :many
SELECT *
FROM payment_events
WHERE payment_id = sqlc.arg(payment_id)
  AND organization_id = sqlc.arg(organization_id)
ORDER BY occurred_at DESC, id DESC;

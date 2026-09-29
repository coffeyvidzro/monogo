-- name: ScheduleNumberRenewals :execrows
INSERT INTO number_renewals (
    organization_id,
    phone_number_id,
    period_start,
    period_end,
    operation_id
)
SELECT
    pn.organization_id,
    pn.id,
    pn.next_renewal_at,
    pn.next_renewal_at + INTERVAL '1 month',
    md5(pn.id::TEXT || ':' || pn.next_renewal_at::TEXT)::UUID
FROM phone_numbers AS pn
WHERE pn.provisioning_mode = 'managed'
  AND pn.status = 'active'
  AND pn.next_renewal_at <= sqlc.arg(now)
ON CONFLICT (phone_number_id, period_start) DO NOTHING;

-- name: ListNumberRenewalsDue :many
SELECT renewal.*, pn.country_code
FROM number_renewals AS renewal
JOIN phone_numbers AS pn ON pn.id = renewal.phone_number_id
WHERE (
    renewal.status IN ('pending', 'payment_failed')
    AND renewal.next_attempt_at <= sqlc.arg(now)
  ) OR (
    renewal.status = 'processing'
    AND renewal.updated_at <= sqlc.arg(now) - INTERVAL '15 minutes'
  )
ORDER BY renewal.next_attempt_at, renewal.period_start
LIMIT sqlc.arg(batch);

-- name: ClaimNumberRenewal :one
UPDATE number_renewals
SET
    status = 'processing',
    attempt_count = attempt_count + 1,
    last_error = NULL
WHERE id = sqlc.arg(id)
  AND (
    status IN ('pending', 'payment_failed')
    OR (status = 'processing' AND updated_at <= sqlc.arg(now) - INTERVAL '15 minutes')
  )
RETURNING *;

-- name: MarkNumberRenewalPaid :one
UPDATE number_renewals AS renewal
SET
    status = 'paid',
    amount_micros = sqlc.arg(amount_micros),
    currency = sqlc.arg(currency),
    paid_at = sqlc.arg(paid_at),
    last_error = NULL
FROM phone_numbers AS pn
WHERE renewal.id = sqlc.arg(id)
  AND renewal.status = 'processing'
  AND pn.id = renewal.phone_number_id
  AND pn.provisioning_mode = 'managed'
  AND pn.status = 'active'
RETURNING renewal.*;

-- name: AdvanceNumberRenewal :execrows
UPDATE phone_numbers
SET
    next_renewal_at = sqlc.arg(period_end),
    updated_at = now()
WHERE id = sqlc.arg(phone_number_id)
  AND provisioning_mode = 'managed'
  AND status = 'active'
  AND next_renewal_at = sqlc.arg(period_start);

-- name: RetryNumberRenewal :one
UPDATE number_renewals
SET
    status = 'payment_failed',
    next_attempt_at = sqlc.arg(next_attempt_at),
    last_error = sqlc.arg(last_error)
WHERE id = sqlc.arg(id)
  AND status = 'processing'
RETURNING *;

-- name: AdvancePaidNumberRenewals :execrows
UPDATE phone_numbers AS pn
SET
    next_renewal_at = renewal.period_end,
    updated_at = now()
FROM number_renewals AS renewal
WHERE renewal.phone_number_id = pn.id
  AND renewal.status = 'paid'
  AND pn.provisioning_mode = 'managed'
  AND pn.status = 'active'
  AND pn.next_renewal_at = renewal.period_start;

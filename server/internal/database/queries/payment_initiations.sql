-- name: ClaimPaymentInitiation :one
WITH available AS MATERIALIZED (
    SELECT p.id, p.checkout_id
    FROM payments AS p
    JOIN checkouts AS c ON c.id = p.checkout_id
    JOIN wallets AS w ON w.id = c.wallet_id
    WHERE p.id = sqlc.arg(payment_id)::UUID
      AND w.organization_id = sqlc.arg(organization_id)::UUID
      AND c.status = 'pending' AND c.expires_at > now()
      AND p.status = 'created' AND p.provider_reference IS NULL
    FOR UPDATE OF c
)
INSERT INTO payment_initiations (
    payment_id, checkout_id, request_hash, provider_reference
)
SELECT a.id, a.checkout_id, sqlc.arg(request_hash)::TEXT,
       sqlc.narg(provider_reference)::TEXT
FROM available AS a
ON CONFLICT DO NOTHING
RETURNING *;

-- name: GetPaymentInitiation :one
SELECT i.* FROM payment_initiations AS i
JOIN payments AS p ON p.id = i.payment_id
JOIN checkouts AS c ON c.id = p.checkout_id
JOIN wallets AS w ON w.id = c.wallet_id
WHERE w.organization_id = sqlc.arg(organization_id)::UUID
  AND i.payment_id = sqlc.arg(payment_id)::UUID LIMIT 1;

-- name: MarkPaymentInitiationSubmitted :one
UPDATE payment_initiations SET
    state = 'submitted',
    provider_reference = sqlc.arg(provider_reference)::TEXT,
    client_secret = sqlc.narg(client_secret)::TEXT,
    display_text = sqlc.narg(display_text)::TEXT,
    upstream_status = sqlc.arg(upstream_status)::TEXT
WHERE payment_id = sqlc.arg(payment_id)::UUID AND state = 'submitting'
  AND (provider_reference IS NULL OR provider_reference = sqlc.arg(provider_reference)::TEXT)
RETURNING *;

-- name: MarkPaymentInitiationUncertain :exec
UPDATE payment_initiations SET state = 'uncertain'
WHERE payment_id = sqlc.arg(payment_id)::UUID AND state = 'submitting';

-- name: SavePaymentProviderReference :one
UPDATE payments SET
    provider_reference = COALESCE(provider_reference, sqlc.arg(provider_reference)::TEXT),
    status = CASE WHEN status = 'created' THEN 'pending' ELSE status END
WHERE id = sqlc.arg(payment_id)::UUID
  AND status IN ('created', 'pending', 'succeeded')
  AND (provider_reference IS NULL OR provider_reference = sqlc.arg(provider_reference)::TEXT)
RETURNING *;

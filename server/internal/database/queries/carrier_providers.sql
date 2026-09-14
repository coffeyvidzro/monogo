-- name: GetCarrierProviderByID :one
SELECT *
FROM carrier_providers
WHERE id = sqlc.arg(id)
  AND status = 'active'
LIMIT 1;

-- name: GetCarrierProviderBySlug :one
SELECT *
FROM carrier_providers
WHERE slug = sqlc.arg(slug)
  AND status = 'active'
LIMIT 1;

-- name: ListCarrierProviders :many
SELECT *
FROM carrier_providers
WHERE status = 'active'
ORDER BY name ASC, slug ASC;


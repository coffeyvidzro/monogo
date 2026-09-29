-- name: ResolveProductRate :one
SELECT pr.*
FROM product_rates AS pr
JOIN organizations AS o
  ON o.id = sqlc.arg(organization_id)
 AND o.status = 'active'
 AND o.deleted_at IS NULL
WHERE (pr.organization_id = o.id OR pr.organization_id IS NULL)
  AND pr.product = sqlc.arg(product)
  AND (pr.selector = sqlc.arg(selector) OR pr.selector = '*')
  AND pr.currency = sqlc.arg(currency)
  AND pr.effective_at <= sqlc.arg(resolved_at)
  AND (pr.expires_at IS NULL OR pr.expires_at > sqlc.arg(resolved_at))
ORDER BY
    (pr.organization_id IS NOT NULL) DESC,
    (pr.selector = sqlc.arg(selector)) DESC,
    pr.effective_at DESC
LIMIT 1;

-- name: GetManagedNumberOrder :one
SELECT * FROM managed_number_orders
WHERE organization_id = sqlc.arg(organization_id) AND id = sqlc.arg(id)
LIMIT 1;

-- name: ListManagedNumberOrdersForReconciliation :many
SELECT * FROM managed_number_orders
WHERE status IN ('submitting', 'outcome_unknown', 'provider_pending', 'configuring')
  AND reconcile_after <= now()
ORDER BY reconcile_after, created_at
LIMIT sqlc.arg(batch_size);

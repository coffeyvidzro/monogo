-- name: CreateManagedReleaseOperation :one
INSERT INTO number_lifecycle_operations (
    organization_id, phone_number_id, provider_id, idempotency_key,
    operation, requested_number, request_payload
)
SELECT pn.organization_id, pn.id, pn.provider_id, sqlc.arg(idempotency_key),
       'release', pn.number, '{}'::JSONB
FROM phone_numbers AS pn
WHERE pn.id = sqlc.arg(phone_number_id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.provisioning_mode = 'managed'
  AND pn.provider_id IS NOT NULL
  AND pn.provider_resource_id IS NOT NULL
  AND pn.status IN ('active', 'disabled')
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetNumberLifecycleOperation :one
SELECT * FROM number_lifecycle_operations
WHERE id = sqlc.arg(id) AND organization_id = sqlc.arg(organization_id);

-- name: GetNumberLifecycleOperationByKey :one
SELECT * FROM number_lifecycle_operations
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key);

-- name: ListNumberLifecycleOperationsDue :many
SELECT * FROM number_lifecycle_operations
WHERE status IN ('pending', 'submitted', 'in_progress')
  AND reconcile_after <= now()
ORDER BY reconcile_after, created_at
LIMIT sqlc.arg(batch_size);

-- name: MarkNumberLifecycleSubmitted :one
UPDATE number_lifecycle_operations
SET status = 'submitted',
    provider_reference = COALESCE(sqlc.narg(provider_reference), provider_reference),
    submitted_at = COALESCE(submitted_at, now()),
    reconcile_after = sqlc.arg(reconcile_after),
    reconcile_attempts = reconcile_attempts + 1
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'submitted', 'in_progress')
RETURNING *;

-- name: ScheduleNumberLifecycleReconciliation :one
UPDATE number_lifecycle_operations
SET status = CASE WHEN status = 'pending' THEN 'submitted' ELSE status END,
    submitted_at = COALESCE(submitted_at, now()),
    reconcile_after = sqlc.arg(reconcile_after),
    reconcile_attempts = reconcile_attempts + 1,
    failure_code = NULL,
    failure_message = NULL
WHERE id = sqlc.arg(id)
  AND status IN ('pending', 'submitted', 'in_progress')
RETURNING *;

-- name: CompleteNumberLifecycleOperation :one
UPDATE number_lifecycle_operations
SET status = 'completed', completed_at = now(), failure_code = NULL, failure_message = NULL
WHERE id = sqlc.arg(id) AND status IN ('pending', 'submitted', 'in_progress')
RETURNING *;

-- name: FailNumberLifecycleOperation :one
UPDATE number_lifecycle_operations
SET status = 'failed', failure_code = sqlc.arg(failure_code),
    failure_message = sqlc.arg(failure_message)
WHERE id = sqlc.arg(id) AND status IN ('pending', 'submitted', 'in_progress')
RETURNING *;

-- name: CreateEmergencyRegistration :one
INSERT INTO emergency_registrations (
    organization_id, phone_number_id, provider_id, idempotency_key, request_hash,
    name, address_line1, address_line2, locality, region, postal_code, country_code
)
SELECT pn.organization_id, pn.id, pn.provider_id, sqlc.arg(idempotency_key), sqlc.arg(request_hash),
       sqlc.arg(name), sqlc.arg(address_line1), sqlc.narg(address_line2),
       sqlc.arg(locality), sqlc.arg(region), sqlc.arg(postal_code), sqlc.arg(country_code)
FROM phone_numbers AS pn
WHERE pn.id = sqlc.arg(phone_number_id)
  AND pn.organization_id = sqlc.arg(organization_id)
  AND pn.provisioning_mode = 'managed'
  AND pn.status = 'active'
  AND pn.provider_id IS NOT NULL
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: GetEmergencyRegistrationByKey :one
SELECT * FROM emergency_registrations
WHERE organization_id = sqlc.arg(organization_id)
  AND idempotency_key = sqlc.arg(idempotency_key);

-- name: GetCurrentEmergencyRegistration :one
SELECT * FROM emergency_registrations
WHERE organization_id = sqlc.arg(organization_id)
  AND phone_number_id = sqlc.arg(phone_number_id)
  AND status IN ('pending', 'validating', 'active')
ORDER BY created_at DESC LIMIT 1;

-- name: DeactivateCurrentEmergencyRegistration :exec
UPDATE emergency_registrations
SET status = 'deactivated', deactivated_at = now()
WHERE organization_id = sqlc.arg(organization_id)
  AND phone_number_id = sqlc.arg(phone_number_id)
  AND status IN ('pending', 'validating', 'active');

-- name: ListEmergencyRegistrationsDue :many
SELECT * FROM emergency_registrations
WHERE status IN ('pending', 'validating') AND reconcile_after <= now()
ORDER BY reconcile_after, created_at LIMIT sqlc.arg(batch_size);

-- name: MarkEmergencyRegistrationValidating :one
UPDATE emergency_registrations
SET status = 'validating', provider_reference = COALESCE(sqlc.narg(provider_reference), provider_reference),
    validation_message = sqlc.narg(validation_message), reconcile_after = sqlc.arg(reconcile_after),
    reconcile_attempts = reconcile_attempts + 1
WHERE id = sqlc.arg(id) AND status IN ('pending', 'validating')
RETURNING *;

-- name: ActivateEmergencyRegistration :one
UPDATE emergency_registrations
SET status = 'active', provider_reference = sqlc.arg(provider_reference),
    validation_message = NULL, activated_at = now()
WHERE id = sqlc.arg(id) AND status IN ('pending', 'validating')
RETURNING *;

-- name: RejectEmergencyRegistration :one
UPDATE emergency_registrations
SET status = 'rejected', validation_message = sqlc.arg(validation_message)
WHERE id = sqlc.arg(id) AND status IN ('pending', 'validating')
RETURNING *;

-- name: CreatePortInOperation :one
INSERT INTO number_lifecycle_operations (
    organization_id, provider_id, idempotency_key, operation, requested_number, request_payload
)
SELECT sqlc.arg(organization_id), cp.id, sqlc.arg(idempotency_key), 'port_in',
       sqlc.arg(requested_number), sqlc.arg(request_payload)::JSONB
FROM carrier_providers AS cp
JOIN organizations AS o ON o.id = sqlc.arg(organization_id)
WHERE cp.slug = 'didww' AND cp.status = 'active'
  AND o.status = 'active' AND o.deleted_at IS NULL
ON CONFLICT (organization_id, idempotency_key) DO NOTHING
RETURNING *;

-- name: CreatePortInCase :one
INSERT INTO port_in_cases (
    organization_id, lifecycle_operation_id, losing_carrier, account_number,
    authorized_name, service_address, desired_port_date
) VALUES (
    sqlc.arg(organization_id), sqlc.arg(lifecycle_operation_id), sqlc.arg(losing_carrier),
    sqlc.arg(account_number), sqlc.arg(authorized_name), sqlc.arg(service_address)::JSONB,
    sqlc.narg(desired_port_date)
)
RETURNING *;

-- name: GetPortInCase :one
SELECT pc.* FROM port_in_cases AS pc
WHERE pc.id = sqlc.arg(id) AND pc.organization_id = sqlc.arg(organization_id);

-- name: GetPortInCaseByOperation :one
SELECT * FROM port_in_cases WHERE lifecycle_operation_id = sqlc.arg(lifecycle_operation_id);

-- name: AddPortInDocument :one
INSERT INTO port_in_documents (
    organization_id, port_in_case_id, document_type, object_key, sha256
) VALUES (
    sqlc.arg(organization_id), sqlc.arg(port_in_case_id), sqlc.arg(document_type),
    sqlc.arg(object_key), sqlc.arg(sha256)
)
RETURNING *;

-- name: ListPortInDocuments :many
SELECT * FROM port_in_documents
WHERE organization_id = sqlc.arg(organization_id) AND port_in_case_id = sqlc.arg(port_in_case_id)
ORDER BY created_at;

-- name: MarkPortInSubmitted :one
UPDATE port_in_cases
SET status = 'submitted', provider_case_reference = sqlc.arg(provider_case_reference)
WHERE id = sqlc.arg(id) AND status IN ('draft', 'checking_portability', 'documents_required')
RETURNING *;

-- name: MarkPortInDocumentsRequired :one
UPDATE port_in_cases SET status = 'documents_required'
WHERE id = sqlc.arg(id) AND status IN ('draft', 'checking_portability')
RETURNING *;

-- name: MarkPortInFOC :one
UPDATE port_in_cases SET status = 'foc_received', foc_at = sqlc.arg(foc_at)
WHERE id = sqlc.arg(id) AND status IN ('submitted', 'in_progress')
RETURNING *;

-- name: ActivatePortInCase :one
UPDATE port_in_cases
SET status = 'activated', activated_at = now()
WHERE id = sqlc.arg(id)
  AND status IN ('submitted', 'in_progress', 'foc_received')
RETURNING *;

-- name: RejectPortInCase :one
UPDATE port_in_cases
SET status = 'rejected', rejection_code = sqlc.arg(rejection_code),
    rejection_message = sqlc.arg(rejection_message)
WHERE id = sqlc.arg(id)
  AND status IN ('draft', 'checking_portability', 'documents_required', 'submitted', 'in_progress')
RETURNING *;

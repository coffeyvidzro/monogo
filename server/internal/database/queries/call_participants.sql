-- name: CreateCallParticipant :one
INSERT INTO call_participants (
    organization_id,
    call_id,
    subscriber_id,
    role,
    address,
    direction,
    state,
    joined_at
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(call_id),
    sqlc.narg(subscriber_id),
    sqlc.arg(role),
    sqlc.narg(address),
    sqlc.narg(direction),
    COALESCE(sqlc.narg(state), 'joining'),
    sqlc.narg(joined_at)
)
RETURNING *;

-- name: GetCallParticipant :one
SELECT *
FROM call_participants
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: ListCallParticipants :many
SELECT *
FROM call_participants
WHERE organization_id = sqlc.arg(organization_id)
  AND call_id = sqlc.arg(call_id)
ORDER BY created_at ASC;

-- name: UpdateCallParticipantState :one
UPDATE call_participants
SET
    state = sqlc.arg(state),
    joined_at = CASE
        WHEN sqlc.arg(state)::text = 'joined' THEN COALESCE(joined_at, NOW())
        ELSE joined_at
    END,
    left_at = CASE
        WHEN sqlc.arg(state)::text IN ('left', 'failed') THEN COALESCE(left_at, NOW())
        ELSE left_at
    END,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
RETURNING *;

-- name: CreateConferenceParticipant :one
INSERT INTO conference_participants (
    organization_id,
    conference_id,
    call_participant_id,
    state,
    muted,
    deaf,
    speaking,
    joined_at
) VALUES (
    sqlc.arg(organization_id),
    sqlc.arg(conference_id),
    sqlc.narg(call_participant_id),
    COALESCE(sqlc.narg(state), 'joined'),
    COALESCE(sqlc.narg(muted), false),
    COALESCE(sqlc.narg(deaf), false),
    COALESCE(sqlc.narg(speaking), false),
    COALESCE(sqlc.narg(joined_at), NOW())
)
RETURNING *;

-- name: GetConferenceParticipant :one
SELECT *
FROM conference_participants
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: ListConferenceParticipants :many
SELECT *
FROM conference_participants
WHERE organization_id = sqlc.arg(organization_id)
  AND conference_id = sqlc.arg(conference_id)
ORDER BY created_at ASC;

-- name: LeaveConferenceParticipant :one
UPDATE conference_participants
SET
    state = 'left',
    left_at = COALESCE(left_at, NOW()),
    muted = false,
    deaf = false,
    speaking = false,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('joining', 'joined')
RETURNING *;

-- name: FailConferenceParticipant :one
UPDATE conference_participants
SET
    state = 'failed',
    left_at = COALESCE(left_at, NOW()),
    muted = false,
    deaf = false,
    speaking = false,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state IN ('joining', 'joined')
RETURNING *;

-- name: SetConferenceParticipantMuted :one
UPDATE conference_participants
SET
    muted = sqlc.arg(muted),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state = 'joined'
  AND muted <> sqlc.arg(muted)
RETURNING *;

-- name: SetConferenceParticipantDeaf :one
UPDATE conference_participants
SET
    deaf = sqlc.arg(deaf),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state = 'joined'
  AND deaf <> sqlc.arg(deaf)
RETURNING *;

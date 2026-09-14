-- name: CreateConference :one
INSERT INTO conferences (
    organization_id,
    application_id,
    name,
    state,
    started_at
) VALUES (
    sqlc.arg(organization_id),
    sqlc.narg(application_id),
    sqlc.arg(name),
    COALESCE(sqlc.narg(state), 'active'),
    COALESCE(sqlc.narg(started_at), NOW())
)
RETURNING *;

-- name: GetConference :one
SELECT *
FROM conferences
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
LIMIT 1;

-- name: GetConferenceByName :one
SELECT *
FROM conferences
WHERE organization_id = sqlc.arg(organization_id)
  AND name = sqlc.arg(name)
LIMIT 1;

-- name: ListConferences :many
SELECT *
FROM conferences
WHERE organization_id = sqlc.arg(organization_id)
ORDER BY created_at DESC
LIMIT sqlc.arg(page_limit)
OFFSET sqlc.arg(page_offset);

-- name: EndConference :one
UPDATE conferences
SET
    state = 'ended',
    ended_at = COALESCE(ended_at, NOW()),
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND id = sqlc.arg(id)
  AND state = 'active'
RETURNING *;

-- name: EndConferenceParticipants :many
UPDATE conference_participants
SET
    state = 'left',
    left_at = COALESCE(left_at, NOW()),
    muted = false,
    deaf = false,
    speaking = false,
    updated_at = NOW()
WHERE organization_id = sqlc.arg(organization_id)
  AND conference_id = sqlc.arg(conference_id)
  AND state IN ('joining', 'joined')
RETURNING *;
-- name: ListBackofficeConferences :many
SELECT c.id::TEXT AS id, c.organization_id::TEXT AS organization_id, o.name AS organization_name,
       c.name, c.state, COUNT(cp.id)::BIGINT AS participant_count,
       CAST(COALESCE(to_char(c.started_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS started_at,
       CAST(COALESCE(to_char(c.ended_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS ended_at
FROM conferences c JOIN organizations o ON o.id=c.organization_id
LEFT JOIN conference_participants cp ON cp.conference_id=c.id
GROUP BY c.id,o.name ORDER BY c.created_at DESC LIMIT 100;

-- name: GetBackofficeConference :one
SELECT c.id::TEXT AS id, c.organization_id::TEXT AS organization_id, o.name AS organization_name,
       CAST(COALESCE(c.application_id::TEXT,'—') AS TEXT) AS application_id,
       COALESCE(va.name,'—') AS application_name, c.name, c.state,
       COUNT(cp.id)::BIGINT AS participant_count,
       COUNT(cp.id) FILTER (WHERE cp.left_at IS NULL)::BIGINT AS active_participant_count,
       CAST(COALESCE(to_char(c.started_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS started_at,
       CAST(COALESCE(to_char(c.ended_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI'),'—') AS TEXT) AS ended_at,
       to_char(c.created_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI') AS created_at,
       to_char(c.updated_at AT TIME ZONE 'UTC','YYYY-MM-DD HH24:MI') AS updated_at
FROM conferences c JOIN organizations o ON o.id=c.organization_id
LEFT JOIN voice_applications va ON va.id=c.application_id LEFT JOIN conference_participants cp ON cp.conference_id=c.id
WHERE c.id=sqlc.arg(id) GROUP BY c.id,o.name,va.name LIMIT 1;

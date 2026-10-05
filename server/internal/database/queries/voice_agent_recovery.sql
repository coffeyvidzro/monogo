-- name: ListCallsForVoiceAgentRecovery :many
SELECT c.*
FROM calls AS c
JOIN organizations AS o ON o.id = c.organization_id
WHERE c.voice_agent_id IS NOT NULL
  AND c.state IN ('answered', 'active')
  AND c.ended_at IS NULL
  AND o.status = 'active'
  AND o.deleted_at IS NULL
ORDER BY c.created_at ASC;

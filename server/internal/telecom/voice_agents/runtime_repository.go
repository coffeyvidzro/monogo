package voice_agents

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

func (r *Repository) CreateSession(ctx context.Context, organizationID, callID, agentID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	return r.queries.CreateVoiceAgentSession(ctx, sqlc.CreateVoiceAgentSessionParams{
		OrganizationID: organizationID,
		CallID:         callID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) GetActiveSessionByCall(ctx context.Context, organizationID, callID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	return r.queries.GetActiveVoiceAgentSessionByCallID(ctx, sqlc.GetActiveVoiceAgentSessionByCallIDParams{
		OrganizationID: organizationID,
		CallID:         callID,
	})
}

func (r *Repository) CompleteSession(ctx context.Context, organizationID, id uuid.UUID, req CompleteSessionRequest) (sqlc.VoiceAgentSession, error) {
	return r.queries.CompleteVoiceAgentSession(ctx, sqlc.CompleteVoiceAgentSessionParams{
		State:                  req.State,
		TurnCount:              req.TurnCount,
		InterruptionCount:      req.InterruptionCount,
		FirstResponseLatencyMs: req.FirstResponseLatencyMS,
		AvgTurnLatencyMs:       req.AverageTurnLatencyMS,
		EndedAt:                pgconv.TimeToTimestamptz(req.EndedAt),
		ID:                     id,
		OrganizationID:         organizationID,
	})
}

func (r *Repository) CreateTurn(ctx context.Context, identity SessionIdentity, req CreateTurnRequest) (sqlc.VoiceAgentTurn, error) {
	return r.queries.CreateVoiceAgentTurn(ctx, sqlc.CreateVoiceAgentTurnParams{
		OrganizationID:  identity.OrganizationID,
		SessionID:       identity.SessionID,
		Sequence:        req.Sequence,
		Role:            req.Role,
		Content:         req.Content,
		ProviderID:      req.ProviderID,
		ToolName:        req.ToolName,
		ToolCallID:      req.ToolCallID,
		Metadata:        []byte(req.Metadata),
		SpeechStartedAt: pgconv.NullableTimestamptz(req.SpeechStartedAt),
		SpeechEndedAt:   pgconv.NullableTimestamptz(req.SpeechEndedAt),
		STTLatencyMs:    req.STTLatencyMS,
		LlmTtftMs:       req.LLMTTFTMS,
		TtsTtfbMs:       req.TTSTTFBMS,
		TurnLatencyMs:   req.TurnLatencyMS,
	})
}

func (r *Repository) ListTurns(ctx context.Context, identity SessionIdentity) ([]sqlc.VoiceAgentTurn, error) {
	return r.queries.ListVoiceAgentTurnsBySessionID(ctx, sqlc.ListVoiceAgentTurnsBySessionIDParams{
		OrganizationID: identity.OrganizationID,
		SessionID:      identity.SessionID,
	})
}

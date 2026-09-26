package conversations

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct{ queries *sqlc.Queries }

func NewRepository(queries *sqlc.Queries) *Repository { return &Repository{queries: queries} }

func (r *Repository) Create(ctx context.Context, organizationID, callID, agentID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	return r.queries.CreateVoiceAgentSession(ctx, sqlc.CreateVoiceAgentSessionParams{
		OrganizationID: organizationID, CallID: callID, VoiceAgentID: agentID,
	})
}

func (r *Repository) GetActiveByCall(ctx context.Context, organizationID, callID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	return r.queries.GetActiveVoiceAgentSessionByCallID(ctx, sqlc.GetActiveVoiceAgentSessionByCallIDParams{
		OrganizationID: organizationID, CallID: callID,
	})
}

func (r *Repository) Complete(ctx context.Context, organizationID, id uuid.UUID, req CompleteRequest) (sqlc.VoiceAgentSession, error) {
	return r.queries.CompleteVoiceAgentSession(ctx, sqlc.CompleteVoiceAgentSessionParams{
		State: req.State,
		TurnCount: req.TurnCount,
		InterruptionCount: req.InterruptionCount,
		FirstResponseLatencyMs: req.FirstResponseLatencyMS,
		AvgTurnLatencyMs: req.AverageTurnLatencyMS,
		EndedAt: pgconv.TimeToTimestamptz(req.EndedAt),
		ID: id,
		OrganizationID: organizationID,
	})
}

func (r *Repository) CreateTurn(ctx context.Context, identity Identity, req CreateTurnRequest) (sqlc.VoiceAgentTurn, error) {
	return r.queries.CreateVoiceAgentTurn(ctx, sqlc.CreateVoiceAgentTurnParams{
		OrganizationID: identity.OrganizationID,
		SessionID: identity.SessionID,
		Sequence: req.Sequence,
		Role: req.Role,
		Content: req.Content,
		ProviderID: req.ProviderID,
		ToolName: req.ToolName,
		ToolCallID: req.ToolCallID,
		Metadata: []byte(req.Metadata),
		SpeechStartedAt: pgconv.NullableTimestamptz(req.SpeechStartedAt),
		SpeechEndedAt: pgconv.NullableTimestamptz(req.SpeechEndedAt),
		STTLatencyMs: req.STTLatencyMS,
		LlmTtftMs: req.LLMTTFTMS,
		TtsTtfbMs: req.TTSTTFBMS,
		TurnLatencyMs: req.TurnLatencyMS,
	})
}

func (r *Repository) ListTurns(ctx context.Context, identity Identity) ([]sqlc.VoiceAgentTurn, error) {
	return r.queries.ListVoiceAgentTurnsBySessionID(ctx, sqlc.ListVoiceAgentTurnsBySessionIDParams{
		OrganizationID: identity.OrganizationID, SessionID: identity.SessionID,
	})
}

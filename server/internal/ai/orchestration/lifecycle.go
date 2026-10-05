package orchestration

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

// AttachCall resolves the Voice Agent attached to a call and creates the durable AI session snapshot. Replays return the
// existing active session so downstream media attachment can be retried
// without creating duplicate conversation history.
func (s *Service) AttachCall(
	ctx context.Context,
	organizationID, callID, voiceAgentID uuid.UUID,
) (sqlc.VoiceAgentSession, bool, error) {
	if organizationID == uuid.Nil || callID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, false, apperror.NewBadRequest(
			"organization_id and call_id are required",
		)
	}
	if voiceAgentID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, false, nil
	}

	active, err := s.conversations.GetActiveByCall(ctx, organizationID, callID)
	if err == nil {
		return active, true, nil
	}
	if !appErrorCode(err, "NOT_FOUND") {
		return sqlc.VoiceAgentSession{}, false, err
	}

	agent, err := s.agents.Get(ctx, organizationID, voiceAgentID)
	if err != nil {
		if appErrorCode(err, "NOT_FOUND") {
			return sqlc.VoiceAgentSession{}, false, nil
		}
		return sqlc.VoiceAgentSession{}, false, err
	}

	created, err := s.conversations.Create(ctx, organizationID, callID, agent.ID)
	if err == nil {
		return created, true, nil
	}
	if !appErrorCode(err, "CONFLICT") {
		return sqlc.VoiceAgentSession{}, false, err
	}

	active, readErr := s.conversations.GetActiveByCall(ctx, organizationID, callID)
	if readErr != nil {
		return sqlc.VoiceAgentSession{}, false, readErr
	}
	return active, true, nil
}

// CompleteCall closes an active durable Voice Agent session while preserving
// counters and latency summaries already accumulated on that session.
func (s *Service) CompleteCall(
	ctx context.Context,
	organizationID, callID uuid.UUID,
	state string,
	endedAt time.Time,
) (sqlc.VoiceAgentSession, bool, error) {
	active, err := s.conversations.GetActiveByCall(ctx, organizationID, callID)
	if err != nil {
		if appErrorCode(err, "NOT_FOUND") {
			return sqlc.VoiceAgentSession{}, false, nil
		}
		return sqlc.VoiceAgentSession{}, false, err
	}
	if endedAt.IsZero() {
		endedAt = time.Now().UTC()
	}

	completed, err := s.conversations.Complete(ctx, organizationID, active.ID, conversations.CompleteRequest{
		State:                  state,
		TurnCount:              active.TurnCount,
		InterruptionCount:      active.InterruptionCount,
		FirstResponseLatencyMS: active.FirstResponseLatencyMs,
		AverageTurnLatencyMS:   active.AvgTurnLatencyMs,
		EndedAt:                endedAt,
	})
	if err != nil {
		if appErrorCode(err, "NOT_FOUND") {
			return sqlc.VoiceAgentSession{}, false, nil
		}
		return sqlc.VoiceAgentSession{}, false, err
	}
	return completed, true, nil
}

func (s *Service) CreateTurn(
	ctx context.Context,
	organizationID, sessionID uuid.UUID,
	req conversations.CreateTurnRequest,
) (sqlc.VoiceAgentTurn, error) {
	return s.conversations.CreateTurn(ctx, conversations.Identity{
		OrganizationID: organizationID,
		SessionID:      sessionID,
	}, req)
}

func (s *Service) CompleteCallWithSummary(
	ctx context.Context,
	organizationID, callID uuid.UUID,
	state string,
	endedAt time.Time,
	summary conversations.CompleteRequest,
) (sqlc.VoiceAgentSession, bool, error) {
	active, err := s.conversations.GetActiveByCall(ctx, organizationID, callID)
	if err != nil {
		if appErrorCode(err, "NOT_FOUND") {
			return sqlc.VoiceAgentSession{}, false, nil
		}
		return sqlc.VoiceAgentSession{}, false, err
	}
	if endedAt.IsZero() {
		endedAt = time.Now().UTC()
	}
	summary.State = state
	summary.EndedAt = endedAt
	completed, err := s.conversations.Complete(ctx, organizationID, active.ID, summary)
	if err != nil {
		if appErrorCode(err, "NOT_FOUND") {
			return sqlc.VoiceAgentSession{}, false, nil
		}
		return sqlc.VoiceAgentSession{}, false, err
	}
	return completed, true, nil
}

func appErrorCode(err error, code string) bool {
	var appErr *apperror.AppError
	return errors.As(err, &appErr) && appErr.Code == code
}

func (s *Service) ListTurns(ctx context.Context, organizationID, sessionID uuid.UUID) ([]sqlc.VoiceAgentTurn, error) {
	return s.conversations.ListTurns(ctx, conversations.Identity{
		OrganizationID: organizationID,
		SessionID:      sessionID,
	})
}

// ActiveSessionForCall reads existing state without creating a session for a
// terminal lifecycle event that never attached media.
func (s *Service) ActiveSessionForCall(ctx context.Context, organizationID, callID uuid.UUID) (sqlc.VoiceAgentSession, bool, error) {
	active, err := s.conversations.GetActiveByCall(ctx, organizationID, callID)
	if appErrorCode(err, "NOT_FOUND") {
		return sqlc.VoiceAgentSession{}, false, nil
	}
	return active, err == nil, err
}

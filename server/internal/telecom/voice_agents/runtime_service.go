package voice_agents

import (
	"context"
	"encoding/json"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func (s *Service) CreateSession(ctx context.Context, organizationID, callID, agentID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	if organizationID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("organization_id is required")
	}
	if callID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("call id is required")
	}
	if agentID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("voice agent id is required")
	}
	session, err := s.repo.CreateSession(ctx, organizationID, callID, agentID)
	if conflict(err) {
		return sqlc.VoiceAgentSession{}, apperror.NewConflict("call already has an active voice agent session")
	}
	return session, readError(err, "create voice agent session")
}

func (s *Service) GetActiveSessionByCall(ctx context.Context, organizationID, callID uuid.UUID) (sqlc.VoiceAgentSession, error) {
	if organizationID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("organization_id is required")
	}
	if callID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("call id is required")
	}
	session, err := s.repo.GetActiveSessionByCall(ctx, organizationID, callID)
	return session, readError(err, "active voice agent session not found")
}

func (s *Service) CompleteSession(ctx context.Context, organizationID, id uuid.UUID, req CompleteSessionRequest) (sqlc.VoiceAgentSession, error) {
	if organizationID == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("organization_id is required")
	}
	if id == uuid.Nil {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("session id is required")
	}
	if req.State != "completed" && req.State != "failed" && req.State != "cancelled" {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("session state must be completed, failed, or cancelled")
	}
	if req.TurnCount < 0 || req.InterruptionCount < 0 {
		return sqlc.VoiceAgentSession{}, apperror.NewBadRequest("session counters cannot be negative")
	}
	if req.EndedAt.IsZero() {
		req.EndedAt = time.Now().UTC()
	}
	session, err := s.repo.CompleteSession(ctx, organizationID, id, req)
	return session, readError(err, "active voice agent session not found")
}

func (s *Service) CreateTurn(ctx context.Context, identity SessionIdentity, req CreateTurnRequest) (sqlc.VoiceAgentTurn, error) {
	if identity.OrganizationID == uuid.Nil {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("organization_id is required")
	}
	if identity.SessionID == uuid.Nil {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("session id is required")
	}
	if req.Sequence <= 0 {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("turn sequence must be positive")
	}
	if req.Role != "user" && req.Role != "assistant" && req.Role != "tool" && req.Role != "system" {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("turn role is invalid")
	}
	req.Content = strings.TrimSpace(req.Content)
	if req.Content == "" {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("turn content is required")
	}
	if len(req.Metadata) == 0 {
		req.Metadata = json.RawMessage(`{}`)
	}
	var metadata map[string]any
	if !json.Valid(req.Metadata) || json.Unmarshal(req.Metadata, &metadata) != nil {
		return sqlc.VoiceAgentTurn{}, apperror.NewBadRequest("turn metadata must be a JSON object")
	}
	turn, err := s.repo.CreateTurn(ctx, identity, req)
	if conflict(err) {
		return sqlc.VoiceAgentTurn{}, apperror.NewConflict("turn sequence already exists")
	}
	return turn, readError(err, "create voice agent turn")
}

func (s *Service) ListTurns(ctx context.Context, identity SessionIdentity) ([]sqlc.VoiceAgentTurn, error) {
	if identity.OrganizationID == uuid.Nil {
		return nil, apperror.NewBadRequest("organization_id is required")
	}
	if identity.SessionID == uuid.Nil {
		return nil, apperror.NewBadRequest("session id is required")
	}
	turns, err := s.repo.ListTurns(ctx, identity)
	if err != nil {
		return nil, apperror.NewInternal("list voice agent turns", err)
	}
	return turns, nil
}

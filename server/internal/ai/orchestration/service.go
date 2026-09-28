// Package orchestration coordinates durable AI state with realtime media execution.
package orchestration

import (
	"context"
	"github.com/coffeyvidzro/monogo/internal/ai/agents"
	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

type Service struct {
	agents        *agents.Service
	conversations *conversations.Service
	tools         *tools.Executor
}

func (s *Service) CreateTurn(
	ctx context.Context,
	identity conversations.Identity,
	req conversations.CreateTurnRequest,
) (sqlc.VoiceAgentTurn, error) {
	return s.conversations.CreateTurn(ctx, identity, req)
}

func NewService(agentService *agents.Service, conversationService *conversations.Service, executor *tools.Executor) *Service {
	return &Service{agents: agentService, conversations: conversationService, tools: executor}
}

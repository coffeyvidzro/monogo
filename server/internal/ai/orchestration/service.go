// Package orchestration coordinates durable AI state with realtime media execution.
package orchestration

import (
	"github.com/coffeyvidzro/monogo/internal/ai/agents"
	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
)

type Service struct {
	agents        *agents.Service
	conversations *conversations.Service
	tools         *tools.Executor
}

func NewService(agentService *agents.Service, conversationService *conversations.Service, executor *tools.Executor) *Service {
	return &Service{agents: agentService, conversations: conversationService, tools: executor}
}

package ai

import (
	"github.com/coffeyvidzro/monogo/internal/ai/agents"
	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

type Module struct {
	Agents        AgentsModule
	Tools         ToolsModule
	Conversations ConversationsModule
	Orchestration *orchestration.Service
}

type AgentsModule struct {
	Repository *agents.Repository
	Service    *agents.Service
	Handler    *agents.Handler
}

type ToolsModule struct {
	Repository *tools.Repository
	Service    *tools.Service
	Handler    *tools.Handler
	Executor   *tools.Executor
}

type ConversationsModule struct {
	Repository *conversations.Repository
	Service    *conversations.Service
}

func New(queries *sqlc.Queries) *Module {
	agentsRepository := agents.NewRepository(queries)
	agentsService := agents.NewService(agentsRepository)

	toolsRepository := tools.NewRepository(queries)
	toolsService := tools.NewService(toolsRepository)
	toolsExecutor := tools.NewExecutor(toolsService)

	conversationsRepository := conversations.NewRepository(queries)
	conversationsService := conversations.NewService(conversationsRepository)

	orchestrator := orchestration.NewService(agentsService, conversationsService, toolsExecutor)

	return &Module{
		Agents: AgentsModule{
			Repository: agentsRepository,
			Service:    agentsService,
			Handler:    agents.NewHandler(agentsService),
		},
		Tools: ToolsModule{
			Repository: toolsRepository,
			Service:    toolsService,
			Handler:    tools.NewHandler(toolsService),
			Executor:   toolsExecutor,
		},
		Conversations: ConversationsModule{
			Repository: conversationsRepository,
			Service:    conversationsService,
		},
		Orchestration: orchestrator,
	}
}

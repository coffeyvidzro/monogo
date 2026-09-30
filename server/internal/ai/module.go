package ai

import (
	"github.com/coffeyvidzro/monogo/internal/ai/agents"
	"github.com/coffeyvidzro/monogo/internal/ai/conversations"
	"github.com/coffeyvidzro/monogo/internal/ai/orchestration"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/security/encryption"
	"github.com/coffeyvidzro/monogo/internal/telephony/calls"
)

type Dependencies struct {
	CredentialCipher *encryption.Cipher
	Calls            *calls.Service
}

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

func New(queries *sqlc.Queries, dependencies ...Dependencies) *Module {
	var deps Dependencies
	if len(dependencies) > 0 {
		deps = dependencies[0]
	}

	agentsRepository := agents.NewRepository(queries)
	agentsService := agents.NewService(agentsRepository)

	toolsRepository := tools.NewRepository(queries)
	toolsService := tools.NewService(toolsRepository, deps.CredentialCipher)
	toolsExecutor := tools.NewExecutor(toolsService, deps.Calls)

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

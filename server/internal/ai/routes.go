package ai

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/ai/agents"
	"github.com/coffeyvidzro/monogo/internal/ai/tools"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, module *Module, organizationAccess func(string) func(http.Handler) http.Handler) {
	agents.RegisterRoutes(router, module.Agents.Handler, organizationAccess("voice-agents"))
	tools.RegisterRoutes(router, module.Tools.Handler, organizationAccess("voice-agents"))
}

package modules

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/leamout/leamout/internal/modules/audit"
	"github.com/leamout/leamout/internal/modules/webhooks"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
) {
	webhooks.RegisterRoutes(
		router,
		module.Webhooks.Handler,
		organizationAccess("webhooks"),
	)

	audit.RegisterRoutes(
		router,
		module.Audit.Handler,
		organizationAccess("audit"),
	)
}

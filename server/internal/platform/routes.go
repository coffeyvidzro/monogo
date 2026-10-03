package platform

import (
	"net/http"

	"github.com/go-chi/chi/v5"

	"github.com/coffeyvidzro/monogo/internal/platform/audit"
	"github.com/coffeyvidzro/monogo/internal/platform/runtimes"
	"github.com/coffeyvidzro/monogo/internal/platform/storage"
	"github.com/coffeyvidzro/monogo/internal/platform/webhooks"
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

	runtimes.RegisterRoutes(
		router,
		module.Runtimes.Handler,
		organizationAccess("runtimes"),
	)

	storage.RegisterRoutes(
		router,
		module.Storage.Handler,
		organizationAccess("storage"),
	)
}

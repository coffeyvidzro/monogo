package billing

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/billing/charging"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
) {
	charging.RegisterRoutes(
		router,
		module.Charging.Handler,
		organizationAccess("billing"),
	)
}

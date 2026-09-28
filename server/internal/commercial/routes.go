package commercial

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/commercial/pricing"
	"github.com/coffeyvidzro/monogo/internal/commercial/subscriptions"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	module *Module,
	organizationAccess func(string) func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	wallets.RegisterRoutes(
		router,
		module.Wallets.Handler,
		organizationAccess("wallets"),
	)

	subscriptions.RegisterRoutes(
		router,
		module.Subscriptions.Handler,
		organizationAccess("subscriptions"),
		idempotency,
	)

	pricing.RegisterRoutes(
		router,
		module.Pricing.Handler,
		organizationAccess("pricing"),
	)
}

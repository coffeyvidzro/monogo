package commercial

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/commercial/plans"
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
	checkout.RegisterRoutes(
		router,
		module.Checkout.Handler,
		organizationAccess("checkout"),
		idempotency,
	)

	wallets.RegisterRoutes(
		router,
		module.Wallets.Handler,
		organizationAccess("wallets"),
	)

	plans.RegisterRoutes(
		router,
		module.Plans.Handler,
		organizationAccess("plans"),
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

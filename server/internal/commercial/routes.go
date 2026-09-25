package commercial

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/commercial/checkout"
	"github.com/coffeyvidzro/monogo/internal/commercial/payments"
	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, module *Module, organizationAccess func(string) func(http.Handler) http.Handler, idempotency func(http.Handler) http.Handler) {
	auth := organizationAccess("commercial-state")
	checkout.RegisterRoutes(router, module.Checkout.Handler, auth, idempotency)
	payments.RegisterRoutes(router, module.Payments.Handler, auth, idempotency)
	wallets.RegisterRoutes(router, module.Wallets.Handler, auth)
}

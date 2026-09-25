package commercial

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/commercial/wallets"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, module *Module, organizationAccess func(string) func(http.Handler) http.Handler) {
	wallets.RegisterRoutes(router, module.Wallets.Handler, organizationAccess("commercial-state"))
}

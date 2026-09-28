package wallets

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/wallets", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/{currency}", handler.GetByCurrency)
		r.Get("/{currency}/ledger", handler.ListLedger)
	})
}

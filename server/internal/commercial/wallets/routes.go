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
	router.Route("/wallet", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/", handler.Get)
		r.Get("/ledger", handler.ListLedger)
	})
}

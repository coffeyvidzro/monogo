package wallets

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/wallets", func(r chi.Router) {
		r.Use(auth)
		r.Get("/", handler.List)
		r.Get("/{wallet_id}", handler.Get)
		r.Get("/{wallet_id}/transactions", handler.ListTransactions)
	})
}

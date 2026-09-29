package payments

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	router.Route("/payments", func(r chi.Router) {
		r.Use(authMiddleware)

		r.With(idempotency).Post("/subscriptions", handler.CreateSubscription)
		r.With(idempotency).Post("/wallet-topups", handler.CreateWalletTopup)
		r.Get("/{id}", handler.Get)
	})
}

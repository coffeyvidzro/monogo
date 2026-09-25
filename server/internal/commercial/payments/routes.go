package payments

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, idempotency func(http.Handler) http.Handler) {
	router.Group(func(r chi.Router) {
		r.Use(auth)
		r.With(idempotency).Post("/checkouts/{checkout_id}/payments", handler.Create)
		r.Get("/payments/{payment_id}", handler.Get)
	})
}

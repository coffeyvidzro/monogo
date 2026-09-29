package checkout

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
	router.Route("/checkouts", func(r chi.Router) {
		r.Use(authMiddleware)

		r.With(idempotency).Post("/", handler.Create)
		r.Get("/{checkout_id}", handler.Get)
		r.With(idempotency).Post("/{checkout_id}/confirm", handler.Confirm)
		r.With(idempotency).Post("/{checkout_id}/continue", handler.Continue)
	})
}

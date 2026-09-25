package checkout

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, idempotency func(http.Handler) http.Handler) {
	router.Route("/checkouts", func(r chi.Router) {
		r.Use(auth)
		r.With(idempotency).Post("/", handler.Create)
		r.Get("/{checkout_id}", handler.Get)
		r.With(idempotency).Post("/{checkout_id}/cancel", handler.Cancel)
	})
}

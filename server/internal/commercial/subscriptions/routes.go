package subscriptions

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
	router.Route("/subscription-plans", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/", handler.ListPlans)
		r.Get("/{plan_id}", handler.GetPlan)
	})

	router.Route("/subscriptions", func(r chi.Router) {
		r.Use(authMiddleware)

		r.With(idempotency).Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/current", handler.Current)
		r.Get("/{id}", handler.Get)
		r.With(idempotency).Patch("/{id}", handler.Update)
		r.With(idempotency).Delete("/{id}", handler.Delete)
	})
}

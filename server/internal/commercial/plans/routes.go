package plans

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/plans", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/", handler.List)
		r.Get("/{plan_id}", handler.Get)
	})
}

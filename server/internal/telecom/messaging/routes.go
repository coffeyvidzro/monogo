package messaging

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
	router.Route("/messages", func(r chi.Router) {
		r.Use(authMiddleware)
		r.With(idempotency).Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
	})
}

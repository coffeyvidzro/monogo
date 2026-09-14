package voice

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, authMiddleware func(http.Handler) http.Handler) {
	router.Route("/voice-applications", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Patch("/{id}", handler.Update)
		r.Delete("/{id}", handler.Delete)

		r.Post("/{id}/bindings", handler.CreateBinding)
		r.Get("/{id}/bindings", handler.ListBindings)
		r.Delete("/{id}/bindings/{binding_id}", handler.DeleteBinding)
	})
}

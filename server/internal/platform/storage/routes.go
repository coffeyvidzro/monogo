package storage

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	auth func(http.Handler) http.Handler,
) {
	router.Route("/storage-integrations", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{storage_integration_id}", handler.Get)
		r.Patch("/{storage_integration_id}", handler.Update)
		r.Delete("/{storage_integration_id}", handler.Delete)
		r.Post("/{storage_integration_id}/test", handler.Test)
	})
}

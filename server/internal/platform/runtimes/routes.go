package runtimes

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	auth func(http.Handler) http.Handler,
) {
	router.Route("/runtimes", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{runtime_id}", handler.Get)
		r.Post("/{runtime_id}/heartbeat", handler.Heartbeat)
	})
}

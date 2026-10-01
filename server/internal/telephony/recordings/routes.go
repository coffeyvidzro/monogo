package recordings

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/recordings", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Get("/{id}/playback", handler.Playback)
		r.Delete("/{id}", handler.Delete)
	})
}

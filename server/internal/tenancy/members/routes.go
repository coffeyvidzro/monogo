package members

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/organizations/{organization_id}/members", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", handler.Add)
		r.Get("/", handler.List)
		r.Get("/{user_id}", handler.Get)
		r.Patch("/{user_id}", handler.Update)
		r.Delete("/{user_id}", handler.Delete)
	})
}

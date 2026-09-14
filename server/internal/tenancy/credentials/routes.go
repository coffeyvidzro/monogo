package credentials

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, authMiddleware func(http.Handler) http.Handler) {
	router.Route("/organizations/{organization_id}/credentials", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{credential_id}", handler.Get)
		r.Patch("/{credential_id}", handler.Update)
		r.Delete("/{credential_id}", handler.Disable)
	})
}

package users

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/users", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/me", handler.GetMe)
		r.Patch("/me", handler.UpdateMe)
		r.Delete("/me", handler.DeleteMe)
	})
}

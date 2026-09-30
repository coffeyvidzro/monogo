package subscribers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/subscribers", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{subscriber_id}", handler.Get)
		r.Patch("/{subscriber_id}", handler.Update)
		r.Delete("/{subscriber_id}", handler.Delete)
		r.Post("/{subscriber_id}/credentials/rotate", handler.Rotate)
	})
}

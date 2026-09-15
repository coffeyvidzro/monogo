package audit

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/audit-events", func(r chi.Router) {
		r.Use(auth)
		r.Get("/", handler.List)
	})
}

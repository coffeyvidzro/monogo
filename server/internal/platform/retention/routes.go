package retention

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, entitlement func(http.Handler) http.Handler) {
	router.Route("/retention-policies", func(r chi.Router) {
		r.Use(auth)
		r.Use(entitlement)
		r.Get("/", handler.List)
		r.Put("/{resource}", handler.Upsert)
		r.Delete("/{resource}", handler.Delete)
	})
}

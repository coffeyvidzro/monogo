package networking

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, entitlement func(http.Handler) http.Handler) {
	router.Route("/network-policies", func(r chi.Router) {
		r.Use(auth)
		r.Use(entitlement)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Patch("/{network_policy_id}", handler.Update)
		r.Delete("/{network_policy_id}", handler.Delete)
	})
}

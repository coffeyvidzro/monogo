package sso

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, entitlement func(http.Handler) http.Handler) {
	router.Route("/sso/connections", func(r chi.Router) {
		r.Use(auth)
		r.Use(entitlement)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Route("/{sso_connection_id}", func(r chi.Router) {
			r.Get("/", handler.Get)
			r.Patch("/", handler.Update)
			r.Delete("/", handler.Delete)
		})
	})
}

package sip_domains

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/sip-domains", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{sip_domain_id}", handler.Get)
		r.Patch("/{sip_domain_id}", handler.Update)
		r.Delete("/{sip_domain_id}", handler.Delete)
	})
}

package scim

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterManagementRoutes(router chi.Router, handler *Handler, auth, entitlement func(http.Handler) http.Handler) {
	router.Route("/scim/tokens", func(r chi.Router) {
		r.Use(auth)
		r.Use(entitlement)
		r.Post("/", handler.CreateToken)
		r.Get("/", handler.ListTokens)
		r.Delete("/{scim_token_id}", handler.RevokeToken)
	})
}

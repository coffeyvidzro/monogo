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

func RegisterProvisioningRoutes(router chi.Router, handler *Handler, tokenAuth, entitlement func(http.Handler) http.Handler) {
	router.Route("/scim/v2", func(r chi.Router) {
		r.Use(tokenAuth)
		r.Use(entitlement)
		r.Route("/Users", func(r chi.Router) {
			r.Post("/", handler.CreateUser)
			r.Get("/", handler.ListUsers)
			r.Get("/{user_id}", handler.GetUser)
			r.Put("/{user_id}", handler.ReplaceUser)
			r.Delete("/{user_id}", handler.DeleteUser)
		})
		r.Route("/Groups", func(r chi.Router) {
			r.Post("/", handler.CreateGroup)
			r.Get("/", handler.ListGroups)
			r.Get("/{group_id}", handler.GetGroup)
			r.Put("/{group_id}", handler.ReplaceGroup)
			r.Delete("/{group_id}", handler.DeleteGroup)
		})
	})
}

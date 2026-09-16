package server

import "github.com/go-chi/chi/v5"

func newRouter(modules *modules) *chi.Mux {
	router := chi.NewRouter()
	registerHealthRoutes(router, modules)
	return router
}

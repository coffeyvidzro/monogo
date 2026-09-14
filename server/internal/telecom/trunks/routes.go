package trunks

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/trunks", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{trunk_id}", handler.Get)
		r.Patch("/{trunk_id}", handler.Update)
		r.Delete("/{trunk_id}", handler.Delete)
		r.Post("/{trunk_id}/credentials/rotate", handler.RotateCredential)
		r.Post("/{trunk_id}/endpoints", handler.CreateEndpoint)
		r.Get("/{trunk_id}/endpoints", handler.ListEndpoints)
		r.Get("/{trunk_id}/endpoints/{endpoint_id}", handler.GetEndpoint)
		r.Patch("/{trunk_id}/endpoints/{endpoint_id}", handler.UpdateEndpoint)
		r.Delete("/{trunk_id}/endpoints/{endpoint_id}", handler.DeleteEndpoint)
	})
}

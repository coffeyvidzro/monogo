package numbers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	auth func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	router.Route("/numbers", func(r chi.Router) {
		r.Use(auth)
		r.Get("/available", handler.SearchAvailable)
		r.With(idempotency).Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{number_id}", handler.Get)
		r.Patch("/{number_id}", handler.Update)
		r.Delete("/{number_id}", handler.Delete)
		r.Put("/{number_id}/carrier-connection", handler.SetCarrierConnection)
	})
}

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
		r.With(idempotency).Post("/byoc", handler.CreateBYOC)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Patch("/{id}", handler.Update)
		r.Patch("/{id}/carrier-connection", handler.SetBYOCConnection)
		r.With(idempotency).Delete("/{id}/byoc", handler.ReleaseBYOC)
	})
}

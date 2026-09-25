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
	additionalRoutes ...func(chi.Router),
) {
	router.Route("/numbers", func(r chi.Router) {
		r.Use(auth)
		r.Get("/available", handler.SearchAvailable)
		r.With(idempotency).Post("/", handler.CreateNumber)
		r.With(idempotency).Post("/purchase", handler.PurchaseNumber)
		r.Get("/orders/{order_id}", handler.GetManagedOrder)
		r.Get("/", handler.List)
		r.Get("/{number_id}", handler.Get)
		r.Patch("/{number_id}", handler.Update)
		r.Patch("/{number_id}/carrier-connection", handler.SetBYOCConnection)
		r.Delete("/{number_id}", handler.Delete)
		for _, register := range additionalRoutes {
			register(r)
		}
	})
}

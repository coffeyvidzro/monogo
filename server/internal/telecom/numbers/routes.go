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
		r.With(idempotency).Post("/", handler.CreateNumber)
		r.With(idempotency).Post("/purchase", handler.PurchaseNumber)
		r.Get("/orders/{order_id}", handler.GetManagedOrder)
		r.Get("/lifecycle/{operation_id}", handler.GetLifecycle)
		r.With(idempotency).Post("/port-ins", handler.CreatePortIn)
		r.Get("/port-ins/{case_id}", handler.GetPortIn)
		r.With(idempotency).Post("/port-ins/{case_id}/documents", handler.AddPortDocument)
		r.Get("/", handler.List)
		r.Get("/{number_id}", handler.Get)
		r.Patch("/{number_id}", handler.Update)
		r.Patch("/{number_id}/carrier-connection", handler.SetBYOCConnection)
		r.With(idempotency).Post("/{number_id}/release", handler.ReleaseManaged)
		r.With(idempotency).Put("/{number_id}/emergency-registration", handler.PutEmergency)
		r.Get("/{number_id}/emergency-registration", handler.GetEmergency)
		r.Delete("/{number_id}", handler.Delete)
	})
}

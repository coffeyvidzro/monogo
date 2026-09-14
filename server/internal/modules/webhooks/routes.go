package webhooks

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/webhooks", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{webhook_id}", handler.Get)
		r.Patch("/{webhook_id}", handler.Update)
		r.Delete("/{webhook_id}", handler.Delete)
		r.Post("/{webhook_id}/rotate-secret", handler.RotateSecret)
		r.Post("/{webhook_id}/test", handler.Test)
		r.Get("/{webhook_id}/deliveries", handler.ListDeliveries)
		r.Get("/{webhook_id}/deliveries/{delivery_id}", handler.GetDelivery)
		r.Post("/{webhook_id}/deliveries/{delivery_id}/retry", handler.Retry)
	})
}

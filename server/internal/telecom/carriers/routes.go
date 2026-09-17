package carriers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth, idempotency func(http.Handler) http.Handler) {
	router.Route("/carrier-connections", func(r chi.Router) {
		r.Use(auth)
		r.With(idempotency).Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{carrier_connection_id}", handler.Get)
		r.Patch("/{carrier_connection_id}", handler.Update)
		r.Put("/{carrier_connection_id}/outbound-auth", handler.SetOutboundAuth)
		r.Put("/{carrier_connection_id}/inbound-auth", handler.SetInboundAuth)
		r.Post("/{carrier_connection_id}/source-ips", handler.AddSourceIP)
		r.Get("/{carrier_connection_id}/source-ips", handler.ListSourceIPs)
		r.Delete("/{carrier_connection_id}/source-ips/{source_ip_id}", handler.DeleteSourceIP)
	})
}

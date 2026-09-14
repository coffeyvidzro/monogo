package carriers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	auth func(http.Handler) http.Handler,
) {
	router.Route("/carrier-connections", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{carrier_connection_id}", handler.Get)
		r.Post("/{carrier_connection_id}/validate", handler.Validate)
		r.Patch("/{carrier_connection_id}", handler.Update)
		r.Delete("/{carrier_connection_id}", handler.Delete)
		r.Put("/{carrier_connection_id}/outbound-auth", handler.SetOutboundAuth)
		r.Delete("/{carrier_connection_id}/outbound-auth", handler.ClearOutboundAuth)
		r.Put("/{carrier_connection_id}/inbound-auth", handler.SetInboundAuth)
		r.Delete("/{carrier_connection_id}/inbound-auth", handler.ClearInboundAuth)
		r.Post("/{carrier_connection_id}/source-ips", handler.CreateSourceIP)
		r.Get("/{carrier_connection_id}/source-ips", handler.ListSourceIPs)
		r.Delete("/{carrier_connection_id}/source-ips/{source_ip_id}", handler.DeleteSourceIP)
	})

	router.Route("/carrier-providers", func(r chi.Router) {
		r.Use(auth)
		r.Get("/", handler.ListProviders)
		r.Get("/{carrier_provider_id}", handler.GetProvider)
	})
}

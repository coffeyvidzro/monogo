package trunks

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	auth func(http.Handler) http.Handler,
) {
	router.Route("/trunks", func(r chi.Router) {
		r.Use(auth)

		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{trunk_id}", handler.Get)
		r.Post("/{trunk_id}/validate", handler.Validate)
		r.Patch("/{trunk_id}", handler.Update)
		r.Delete("/{trunk_id}", handler.Delete)

		r.Put("/{trunk_id}/outbound-auth", handler.SetOutboundAuth)
		r.Delete("/{trunk_id}/outbound-auth", handler.ClearOutboundAuth)
		r.Put("/{trunk_id}/inbound-auth", handler.SetInboundAuth)

		r.Post("/{trunk_id}/source-ips", handler.CreateSourceIP)
		r.Get("/{trunk_id}/source-ips", handler.ListSourceIPs)
		r.Delete(
			"/{trunk_id}/source-ips/{source_ip_id}",
			handler.DeleteSourceIP,
		)

		r.Post("/{trunk_id}/endpoints", handler.CreateEndpoint)
		r.Get("/{trunk_id}/endpoints", handler.ListEndpoints)
		r.Get(
			"/{trunk_id}/endpoints/{endpoint_id}",
			handler.GetEndpoint,
		)
		r.Patch(
			"/{trunk_id}/endpoints/{endpoint_id}",
			handler.UpdateEndpoint,
		)
		r.Delete(
			"/{trunk_id}/endpoints/{endpoint_id}",
			handler.DeleteEndpoint,
		)
	})
}

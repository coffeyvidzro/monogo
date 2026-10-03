package providers

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/ai-provider-credentials", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.CreateCredential)
		r.Get("/", handler.ListCredentials)
		r.Post("/{credential_id}/rotate", handler.RotateCredential)
		r.Delete("/{credential_id}", handler.DeleteCredential)
	})
	router.Route("/ai-integrations", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.CreateCredential)
		r.Get("/", handler.ListIntegrations)
		r.Post("/{integration_id}/verify", handler.VerifyIntegration)
		r.Post("/{integration_id}/rotate", handler.RotateCredential)
		r.Delete("/{integration_id}", handler.DeleteCredential)
	})
	router.Route("/voice-agents/{voice_agent_id}/providers", func(r chi.Router) {
		r.Use(auth)
		r.Get("/", handler.ListBindings)
		r.Put("/{role}", handler.UpsertBinding)
		r.Delete("/{role}", handler.DeleteBinding)
	})
}

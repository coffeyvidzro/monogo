package agents

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/voice-agents", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{voice_agent_id}", handler.Get)
		r.Patch("/{voice_agent_id}", handler.Update)
		r.Delete("/{voice_agent_id}", handler.Delete)
		r.Post("/{voice_agent_id}/bindings", handler.CreateBinding)
		r.Get("/{voice_agent_id}/bindings", handler.ListBindings)
		r.Delete("/{voice_agent_id}/bindings/{binding_id}", handler.DeleteBinding)
	})
}

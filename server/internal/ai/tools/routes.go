package tools

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, auth func(http.Handler) http.Handler) {
	router.Route("/voice-agents/{voice_agent_id}/tools", func(r chi.Router) {
		r.Use(auth)
		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Patch("/{tool_id}", handler.Update)
		r.Delete("/{tool_id}", handler.Delete)
	})
}

package pricing

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/voice-rates", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Get("/resolve", handler.ResolveVoiceRate)
	})
}

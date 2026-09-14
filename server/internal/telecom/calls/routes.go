package calls

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/calls", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", handler.Create)
		r.Get("/", handler.List)

		r.Get("/{id}", handler.Get)
		r.Post("/{id}/answer", handler.Answer)
		r.Post("/{id}/hangup", handler.Hangup)
		r.Post("/{id}/transfer", handler.Transfer)
		r.Post("/{id}/hold", handler.Hold)
		r.Post("/{id}/unhold", handler.Unhold)
		r.Post("/{id}/play", handler.Play)
		r.Post("/{id}/stop", handler.Stop)
		r.Post("/{id}/record", handler.Record)
		r.Post("/{id}/dtmf", handler.DTMF)
	})
}

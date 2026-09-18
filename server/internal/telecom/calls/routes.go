package calls

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
	idempotency func(http.Handler) http.Handler,
) {
	router.Route("/calls", func(r chi.Router) {
		r.Use(authMiddleware)

		r.With(idempotency).Post("/", handler.Create)
		r.Get("/", handler.List)

		r.Get("/{id}", handler.Get)
		r.With(idempotency).Post("/{id}/answer", handler.Answer)
		r.With(idempotency).Post("/{id}/hangup", handler.Hangup)
		r.With(idempotency).Post("/{id}/transfer", handler.Transfer)
		r.With(idempotency).Post("/{id}/hold", handler.Hold)
		r.With(idempotency).Post("/{id}/unhold", handler.Unhold)
		r.With(idempotency).Post("/{id}/play", handler.Play)
		r.With(idempotency).Post("/{id}/stop", handler.Stop)
		r.With(idempotency).Post("/{id}/record", handler.Record)
		r.With(idempotency).Post("/{id}/dtmf", handler.DTMF)
	})
}

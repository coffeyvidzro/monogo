package conferences

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/conferences", func(r chi.Router) {
		r.Use(authMiddleware)

		r.Post("/", handler.Create)
		r.Get("/", handler.List)
		r.Get("/{id}", handler.Get)
		r.Delete("/{id}", handler.End)
		r.Get("/{id}/participants", handler.ListParticipants)
		r.Post("/{id}/participants", handler.AddParticipant)
		r.Delete("/{id}/participants/{participant_id}", handler.RemoveParticipant)
		r.Post("/{id}/participants/{participant_id}/mute", handler.Mute)
		r.Post("/{id}/participants/{participant_id}/unmute", handler.Unmute)
		r.Post("/{id}/participants/{participant_id}/deaf", handler.Deaf)
		r.Post("/{id}/participants/{participant_id}/undeaf", handler.Undeaf)
		r.Post("/{id}/participants/{participant_id}/kick", handler.Kick)
		r.Post("/{id}/lock", handler.Lock)
		r.Post("/{id}/unlock", handler.Unlock)
	})
}

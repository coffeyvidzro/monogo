package webrtc

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(router chi.Router, handler *Handler, authMiddleware func(http.Handler) http.Handler) {
	router.Route("/webrtc", func(r chi.Router) {
		r.Use(authMiddleware)
		r.Post("/ice-credentials", handler.IssueICECredentials)
	})
}

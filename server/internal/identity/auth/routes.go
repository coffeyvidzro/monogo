package auth

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authMiddleware func(http.Handler) http.Handler,
) {
	router.Route("/auth", func(r chi.Router) {
		r.Post("/start", handler.Start)

		r.Post("/password/login", handler.LoginWithPassword)

		r.Post("/otp/send", handler.SendOTP)
		r.Post("/otp/verify", handler.VerifyOTP)

		r.Group(func(r chi.Router) {
			r.Use(authMiddleware)

			r.Post("/password/enroll", handler.SetPassword)
		})
	})
}

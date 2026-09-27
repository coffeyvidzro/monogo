package charging

import (
	"net/http"

	"github.com/go-chi/chi/v5"
)

func RegisterRoutes(
	router chi.Router,
	handler *Handler,
	authorize func(http.Handler) http.Handler,
) {
	router.Group(func(r chi.Router) {
		r.Use(authorize)

		r.Post("/wallets/{wallet_id}/credits", handler.Credit)
		r.Post("/charges/{charge_id}/reserve", handler.Reserve)
		r.Post("/charges/{charge_id}/consume", handler.Consume)
		r.Post("/charges/{charge_id}/release", handler.Release)
		r.Post("/charges/{charge_id}/debit", handler.Debit)
		r.Post("/charges/{charge_id}/finalize", handler.Finalize)
	})
}

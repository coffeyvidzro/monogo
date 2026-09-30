package checkout

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
	router.With(
		authMiddleware,
		idempotency,
	).Post("/checkouts", handler.Create)

	router.With(
		authMiddleware,
	).Get("/checkouts/{checkout_id}", handler.Get)

	router.With(
		authMiddleware,
		idempotency,
	).Post("/checkouts/{checkout_id}/confirm", handler.Confirm)

}

package lifecycle

import (
	"net/http"
	"github.com/go-chi/chi/v5"
)

// RegisterRoutes attaches lifecycle endpoints to the authenticated /numbers router.
func RegisterRoutes(r chi.Router, handler *Handler, idempotency func(http.Handler) http.Handler) {
	r.Get("/lifecycle/{operation_id}", handler.GetLifecycle)
	r.With(idempotency).Post("/port-ins", handler.CreatePortIn)
	r.Get("/port-ins/{case_id}", handler.GetPortIn)
	r.With(idempotency).Post("/port-ins/{case_id}/documents", handler.AddPortDocument)
	r.With(idempotency).Post("/{number_id}/release", handler.ReleaseManaged)
	r.With(idempotency).Put("/{number_id}/emergency-registration", handler.PutEmergency)
	r.Get("/{number_id}/emergency-registration", handler.GetEmergency)
}

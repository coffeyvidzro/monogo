package retention

import (
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/helper"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler {
	return &Handler{
		service: service,
	}
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	id, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	values, err := h.service.List(r.Context(), id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{
		"retention_policies": values,
	})
}
func (h *Handler) Upsert(w http.ResponseWriter, r *http.Request) {
	id, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[UpsertRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.Upsert(r.Context(), id, chi.URLParam(r, "resource"), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, value)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	id, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), id, chi.URLParam(r, "resource")); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func organizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

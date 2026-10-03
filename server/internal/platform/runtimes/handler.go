package runtimes

import (
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/helper"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.Create(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, response(value, time.Now().UTC()))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	values, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	now := time.Now().UTC()
	out := make([]Response, 0, len(values))
	for _, value := range values {
		out = append(out, response(value, now))
	}
	httputil.OK(w, map[string]any{"runtimes": out})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, runtimeID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.Get(r.Context(), organizationID, runtimeID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(value, time.Now().UTC()))
}

func (h *Handler) Heartbeat(w http.ResponseWriter, r *http.Request) {
	organizationID, runtimeID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[HeartbeatRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.Heartbeat(r.Context(), organizationID, runtimeID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(value, time.Now().UTC()))
}

func organizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

func ids(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := organizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	runtimeID, err := uuid.Parse(chi.URLParam(r, "runtime_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid runtime_id")
	}
	return organizationID, runtimeID, nil
}

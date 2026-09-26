package agents

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

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

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
	agent, err := h.service.Create(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, response(agent))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	out := make([]Response, 0, len(items))
	for _, item := range items {
		out = append(out, response(item))
	}
	httputil.OK(w, map[string]any{"voice_agents": out})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	agent, err := h.service.Get(r.Context(), organizationID, agentID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(agent))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	agent, err := h.service.Update(r.Context(), organizationID, agentID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(agent))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.Disable(r.Context(), organizationID, agentID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[CreateBindingRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	binding, err := h.service.CreateBinding(r.Context(), organizationID, agentID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, bindingResponse(binding))
}

func (h *Handler) ListBindings(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items, err := h.service.ListBindings(r.Context(), organizationID, agentID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	out := make([]BindingResponse, 0, len(items))
	for _, item := range items {
		out = append(out, bindingResponse(item))
	}
	httputil.OK(w, map[string]any{"bindings": out})
}

func (h *Handler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := ids(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	bindingID, err := uuid.Parse(chi.URLParam(r, "binding_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid binding_id"))
		return
	}
	if err := h.service.DeleteBinding(r.Context(), organizationID, agentID, bindingID); err != nil {
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

func ids(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := organizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	agentID, err := uuid.Parse(chi.URLParam(r, "voice_agent_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid voice_agent_id")
	}
	return organizationID, agentID, nil
}

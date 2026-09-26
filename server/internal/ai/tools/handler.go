package tools

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
	orgID, agentID, err := ids(r); if err != nil { httputil.Error(w, err); return }
	req, err := helper.DecodeJSON[CreateRequest](r); if err != nil { httputil.Error(w, err); return }
	tool, err := h.service.Create(r.Context(), orgID, agentID, req); if err != nil { httputil.Error(w, err); return }
	httputil.Created(w, response(tool))
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	orgID, agentID, err := ids(r); if err != nil { httputil.Error(w, err); return }
	items, err := h.service.List(r.Context(), orgID, agentID); if err != nil { httputil.Error(w, err); return }
	out := make([]Response,0,len(items)); for _, item := range items { out = append(out, response(item)) }
	httputil.OK(w, map[string]any{"tools":out})
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	orgID, agentID, err := ids(r); if err != nil { httputil.Error(w, err); return }
	toolID, err := uuid.Parse(chi.URLParam(r,"tool_id")); if err != nil { httputil.Error(w, apperror.NewBadRequest("invalid tool_id")); return }
	req, err := helper.DecodeJSON[UpdateRequest](r); if err != nil { httputil.Error(w, err); return }
	tool, err := h.service.Update(r.Context(), orgID, agentID, toolID, req); if err != nil { httputil.Error(w, err); return }
	httputil.OK(w, response(tool))
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	orgID, agentID, err := ids(r); if err != nil { httputil.Error(w, err); return }
	toolID, err := uuid.Parse(chi.URLParam(r,"tool_id")); if err != nil { httputil.Error(w, apperror.NewBadRequest("invalid tool_id")); return }
	if err := h.service.Delete(r.Context(), orgID, agentID, toolID); err != nil { httputil.Error(w, err); return }
	w.WriteHeader(http.StatusNoContent)
}
func ids(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	orgID, ok := middleware.OrganizationIDFromContext(r.Context()); if !ok { return uuid.Nil,uuid.Nil,apperror.NewBadRequest("organization context required") }
	agentID, err := uuid.Parse(chi.URLParam(r,"voice_agent_id")); if err != nil { return uuid.Nil,uuid.Nil,apperror.NewBadRequest("invalid voice_agent_id") }
	return orgID,agentID,nil
}

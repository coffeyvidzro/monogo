package providers

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

func (h *Handler) CreateCredential(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[CreateCredentialRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.CreateCredential(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, credentialResponse(value, []uuid.UUID{}))
}

func (h *Handler) ListCredentials(w http.ResponseWriter, r *http.Request) {
	h.listIntegrations(w, r, "provider_credentials")
}

func (h *Handler) ListIntegrations(w http.ResponseWriter, r *http.Request) {
	h.listIntegrations(w, r, "ai_integrations")
}

func (h *Handler) listIntegrations(
	w http.ResponseWriter,
	r *http.Request,
	responseKey string,
) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	values, err := h.service.ListIntegrations(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	out := make([]CredentialResponse, 0, len(values))
	for _, value := range values {
		out = append(out, credentialResponse(value.Credential, value.VoiceAgentIDs))
	}
	httputil.OK(w, map[string]any{
		responseKey: out,
	})
}

func (h *Handler) RotateCredential(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := integrationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[RotateCredentialRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.RotateCredential(r.Context(), organizationID, id, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, credentialResponse(value, []uuid.UUID{}))
}

func (h *Handler) VerifyIntegration(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "integration_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid integration_id"))
		return
	}
	value, err := h.service.VerifyIntegration(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, VerificationResponse{
		ID:              value.ID,
		ConnectionState: value.ConnectionState,
		VerifiedAt:      value.VerifiedAt,
		FailureCode:     value.FailureCode,
	})
}

func (h *Handler) DeleteCredential(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := integrationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.DeleteCredential(r.Context(), organizationID, id); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) UpsertBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := agentIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[UpsertBindingRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.UpsertBinding(r.Context(), organizationID, agentID, chi.URLParam(r, "role"), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, bindingResponse(value))
}

func (h *Handler) ListBindings(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := agentIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	values, err := h.service.ListBindings(r.Context(), organizationID, agentID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	out := make([]BindingResponse, 0, len(values))
	for _, value := range values {
		out = append(out, bindingResponse(value))
	}
	httputil.OK(w, map[string]any{"providers": out})
}

func (h *Handler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, agentID, err := agentIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.DeleteBinding(r.Context(), organizationID, agentID, chi.URLParam(r, "role")); err != nil {
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

func agentIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
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

func integrationID(r *http.Request) (uuid.UUID, error) {
	raw := chi.URLParam(r, "integration_id")
	if raw == "" {
		raw = chi.URLParam(r, "credential_id")
	}
	id, err := uuid.Parse(raw)
	if err != nil {
		return uuid.Nil, apperror.NewBadRequest("invalid integration_id")
	}
	return id, nil
}

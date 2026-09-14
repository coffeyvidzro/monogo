package voice

import (
	"encoding/json"
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/platform/middleware"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
		return
	}

	var req CreateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return
	}

	app, err := h.service.Create(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, applicationResponse(app))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
		return
	}

	apps, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]ApplicationResponse, 0, len(apps))
	for _, app := range apps {
		items = append(items, applicationResponse(app))
	}
	httputil.OK(w, map[string]any{"voice_applications": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	app, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, applicationResponse(app))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var req UpdateApplicationRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return
	}

	app, err := h.service.Update(r.Context(), organizationID, id, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, applicationResponse(app))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Disable(r.Context(), organizationID, id); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, map[string]string{"message": "voice application disabled"})
}

func (h *Handler) CreateBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, applicationID, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var req CreateBindingRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return
	}

	binding, err := h.service.CreateBinding(r.Context(), organizationID, applicationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, bindingResponse(binding))
}

func (h *Handler) ListBindings(w http.ResponseWriter, r *http.Request) {
	organizationID, applicationID, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	bindings, err := h.service.ListBindings(r.Context(), organizationID, applicationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]BindingResponse, 0, len(bindings))
	for _, binding := range bindings {
		items = append(items, bindingResponse(binding))
	}
	httputil.OK(w, map[string]any{"bindings": items})
}

func (h *Handler) DeleteBinding(w http.ResponseWriter, r *http.Request) {
	organizationID, applicationID, err := requestApplicationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	bindingID, err := uuid.Parse(chi.URLParam(r, "binding_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid binding_id"))
		return
	}

	if err := h.service.DeleteBinding(r.Context(), organizationID, applicationID, bindingID); err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, map[string]string{"message": "voice binding deleted"})
}

func requestApplicationIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid voice application id")
	}

	return organizationID, id, nil
}

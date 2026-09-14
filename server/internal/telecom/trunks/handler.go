package trunks

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/platform/middleware"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/helper"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	org, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	result, err := h.service.Create(r.Context(), org, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if result.Credential != nil {
		w.Header().Set("Cache-Control", "no-store")
		httputil.Created(w, ManagedCreateResponse{Response: response(result.Trunk), SIP: *result.Credential})
		return
	}
	httputil.Created(w, response(result.Trunk))
}

func (h *Handler) RotateCredential(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	credential, err := h.service.RotateCredential(r.Context(), org, trunk)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	httputil.OK(w, map[string]any{"sip": credential})
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	org, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items, err := h.service.List(r.Context(), org)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	result := make([]Response, 0, len(items))
	for _, item := range items {
		result = append(result, response(item))
	}
	httputil.OK(w, map[string]any{"trunks": result})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	item, err := h.service.Get(r.Context(), org, trunk)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(item))
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	item, err := h.service.Update(r.Context(), org, trunk, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(item))
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), org, trunk); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[EndpointCreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	item, err := h.service.CreateEndpoint(r.Context(), org, trunk, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, endpointResponse(item))
}
func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items, err := h.service.ListEndpoints(r.Context(), org, trunk)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	result := make([]EndpointResponse, 0, len(items))
	for _, item := range items {
		result = append(result, endpointResponse(item))
	}
	httputil.OK(w, map[string]any{"endpoints": result})
}
func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	org, trunk, endpoint, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	item, err := h.service.GetEndpoint(r.Context(), org, trunk, endpoint)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, endpointResponse(item))
}
func (h *Handler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	org, trunk, endpoint, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[EndpointUpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	item, err := h.service.UpdateEndpoint(r.Context(), org, trunk, endpoint, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, endpointResponse(item))
}
func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	org, trunk, endpoint, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.DeleteEndpoint(r.Context(), org, trunk, endpoint); err != nil {
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
func trunkIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	org, err := organizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "trunk_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid trunk_id")
	}
	return org, id, nil
}
func endpointIDs(r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	org, trunk, err := trunkIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "endpoint_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid endpoint_id")
	}
	return org, trunk, id, nil
}

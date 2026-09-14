package webhooks

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/platform/middleware"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(s *Service) *Handler { return &Handler{s} }
func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	org, ok := organization(w, r)
	if !ok {
		return
	}
	var req CreateRequest
	if !decode(w, r, &req) {
		return
	}
	v, secret, e := h.service.Create(r.Context(), org, req)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.Created(w, map[string]any{"webhook": endpointResponse(v), "signing_secret": encodeSecret(secret)})
}
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	org, ok := organization(w, r)
	if !ok {
		return
	}
	v, e := h.service.List(r.Context(), org)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	items := make([]EndpointResponse, 0, len(v))
	for _, x := range v {
		items = append(items, endpointResponse(x))
	}
	httputil.OK(w, map[string]any{"webhooks": items})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	v, e := h.service.Get(r.Context(), org, id)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, endpointResponse(v))
}
func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	var req UpdateRequest
	if !decode(w, r, &req) {
		return
	}
	v, e := h.service.Update(r.Context(), org, id, req)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, endpointResponse(v))
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	if e := h.service.Delete(r.Context(), org, id); e != nil {
		httputil.Error(w, e)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}
func (h *Handler) RotateSecret(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	v, secret, e := h.service.RotateSecret(r.Context(), org, id)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, map[string]any{"webhook": endpointResponse(v), "signing_secret": encodeSecret(secret)})
}
func (h *Handler) Test(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	status, e := h.service.Test(r.Context(), org, id)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, map[string]int{"response_status": status})
}
func (h *Handler) ListDeliveries(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	limit, offset, e := page(r)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	v, e := h.service.ListDeliveries(r.Context(), org, id, limit, offset)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	items := make([]DeliveryResponse, 0, len(v))
	for _, x := range v {
		items = append(items, deliveryResponse(x))
	}
	httputil.OK(w, map[string]any{"deliveries": items})
}
func (h *Handler) GetDelivery(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	did, e := uuid.Parse(chi.URLParam(r, "delivery_id"))
	if e != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid delivery_id"))
		return
	}
	v, e := h.service.GetDelivery(r.Context(), org, id, did)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, deliveryResponse(v))
}
func (h *Handler) Retry(w http.ResponseWriter, r *http.Request) {
	org, id, ok := ids(w, r)
	if !ok {
		return
	}
	did, e := uuid.Parse(chi.URLParam(r, "delivery_id"))
	if e != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid delivery_id"))
		return
	}
	v, e := h.service.Retry(r.Context(), org, id, did)
	if e != nil {
		httputil.Error(w, e)
		return
	}
	httputil.OK(w, deliveryResponse(v))
}
func organization(w http.ResponseWriter, r *http.Request) (uuid.UUID, bool) {
	v, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
	}
	return v, ok
}
func ids(w http.ResponseWriter, r *http.Request) (uuid.UUID, uuid.UUID, bool) {
	org, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
		return uuid.Nil, uuid.Nil, false
	}
	id, e := uuid.Parse(chi.URLParam(r, "webhook_id"))
	if e != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid webhook_id"))
		return uuid.Nil, uuid.Nil, false
	}
	return org, id, true
}
func decode(w http.ResponseWriter, r *http.Request, v any) bool {
	if json.NewDecoder(r.Body).Decode(v) != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return false
	}
	return true
}
func page(r *http.Request) (int32, int32, error) {
	limit := int32(50)
	offset := int32(0)
	var e error
	if s := r.URL.Query().Get("limit"); s != "" {
		n, x := strconv.ParseInt(s, 10, 32)
		e = x
		limit = int32(n)
	}
	if e == nil && r.URL.Query().Get("offset") != "" {
		n, x := strconv.ParseInt(r.URL.Query().Get("offset"), 10, 32)
		e = x
		offset = int32(n)
	}
	if e != nil {
		return 0, 0, apperror.NewBadRequest("invalid pagination")
	}
	return limit, offset, nil
}

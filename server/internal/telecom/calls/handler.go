package calls

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

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	org, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var req CreateCallRequest
	if !decode(w, r, &req) {
		return
	}

	call, err := h.service.Create(r.Context(), org, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, callResponse(call))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	org, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	limit, offset, state, err := listParams(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	calls, err := h.service.List(r.Context(), org, state, offset, limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]CallResponse, 0, len(calls))
	for _, call := range calls {
		items = append(items, callResponse(call))
	}

	httputil.OK(w, map[string]any{"calls": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	call, err := h.service.Get(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, callResponse(call))
}

func (h *Handler) Answer(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	call, err := h.service.Answer(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, callResponse(call))
}

func (h *Handler) Hangup(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	call, err := h.service.Hangup(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, callResponse(call))
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	var req TransferRequest

	h.control(w, r, &req, func(org, id uuid.UUID) error {
		return h.service.Transfer(r.Context(), org, id, req)
	})
}

func (h *Handler) Hold(w http.ResponseWriter, r *http.Request) {
	h.control(w, r, nil, func(org, id uuid.UUID) error {
		return h.service.Hold(r.Context(), org, id)
	})
}

func (h *Handler) Unhold(w http.ResponseWriter, r *http.Request) {
	h.control(w, r, nil, func(org, id uuid.UUID) error {
		return h.service.Unhold(r.Context(), org, id)
	})
}

func (h *Handler) Play(w http.ResponseWriter, r *http.Request) {
	var req PlayRequest

	h.control(w, r, &req, func(org, id uuid.UUID) error {
		return h.service.Play(r.Context(), org, id, req)
	})
}

func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	h.control(w, r, nil, func(org, id uuid.UUID) error {
		return h.service.Stop(r.Context(), org, id)
	})
}

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	var req RecordRequest

	h.control(w, r, &req, func(org, id uuid.UUID) error {
		return h.service.Record(r.Context(), org, id, req)
	})
}

func (h *Handler) DTMF(w http.ResponseWriter, r *http.Request) {
	var req DTMFRequest

	h.control(w, r, &req, func(org, id uuid.UUID) error {
		return h.service.DTMF(r.Context(), org, id, req)
	})
}

func (h *Handler) control(
	w http.ResponseWriter,
	r *http.Request,
	body any,
	action func(uuid.UUID, uuid.UUID) error,
) {
	org, id, err := requestCallIDs(r)
	if err == nil && body != nil && !decode(w, r, body) {
		return
	}

	if err == nil {
		err = action(org, id)
	}

	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, map[string]string{
		"message": "call control action completed",
	})
}

func decode(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return false
	}

	return true
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

func requestCallIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	org, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid call id")
	}

	return org, id, nil
}

func listParams(r *http.Request) (int32, int32, *string, error) {
	limit, offset := int32(50), int32(0)

	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, nil, apperror.NewBadRequest("invalid limit")
		}

		limit = int32(parsed)
	}

	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, nil, apperror.NewBadRequest("invalid offset")
		}

		offset = int32(parsed)
	}

	var state *string
	if value := r.URL.Query().Get("state"); value != "" {
		state = &value
	}

	return limit, offset, state, nil
}

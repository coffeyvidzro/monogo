package calls

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
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
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var req CreateRequest
	if !decodeCallRequest(w, r, &req) {
		return
	}

	call, err := h.service.Create(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, callResponse(call))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := listRequest(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.List(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	response := make([]CallResponse, 0, len(items))
	for _, call := range items {
		response = append(response, callResponse(call))
	}
	httputil.OK(w, map[string]any{"calls": response})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	call, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, callResponse(call))
}

func (h *Handler) Answer(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, "call answer action completed", h.service.Answer)
}

func (h *Handler) Hangup(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, "call hangup action completed", h.service.Hangup)
}

func (h *Handler) Hold(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, "call hold action completed", h.service.Hold)
}

func (h *Handler) Unhold(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, "call unhold action completed", h.service.Resume)
}

func (h *Handler) Stop(w http.ResponseWriter, r *http.Request) {
	h.simpleAction(w, r, "call playback stop action completed", h.service.Stop)
}

func (h *Handler) Transfer(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	var req TransferActionRequest
	if !decodeCallRequest(w, r, &req) {
		return
	}
	if err := h.service.Transfer(r.Context(), organizationID, id, req); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": "call transfer action completed"})
}

func (h *Handler) Play(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	var req PlayActionRequest
	if !decodeCallRequest(w, r, &req) {
		return
	}
	if err := h.service.Play(r.Context(), organizationID, id, req); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": "call playback action completed"})
}

func (h *Handler) Record(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	var req RecordActionRequest
	if !decodeCallRequest(w, r, &req) {
		return
	}
	if err := h.service.Record(r.Context(), organizationID, id, req); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": "call record action completed"})
}

func (h *Handler) DTMF(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestCallIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	var req DTMFActionRequest
	if !decodeCallRequest(w, r, &req) {
		return
	}
	if err := h.service.DTMF(r.Context(), organizationID, id, req); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": "call DTMF action completed"})
}

func (h *Handler) simpleAction(
	w http.ResponseWriter,
	r *http.Request,
	message string,
	action func(context.Context, uuid.UUID, uuid.UUID) error,
) {
	organizationID, id, err := requestCallIDs(r)
	if err == nil {
		err = action(r.Context(), organizationID, id)
	}
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": message})
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return organizationID, nil
}

func requestCallIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid call id")
	}
	return organizationID, id, nil
}

func decodeCallRequest(w http.ResponseWriter, r *http.Request, value any) bool {
	if err := json.NewDecoder(r.Body).Decode(value); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return false
	}
	return true
}

func listRequest(r *http.Request) (ListRequest, error) {
	req := ListRequest{Offset: 0, Limit: 50}
	if state := strings.TrimSpace(r.URL.Query().Get("state")); state != "" {
		req.State = &state
	}
	var err error
	req.Offset, err = parseInt32Query(r, "offset", req.Offset)
	if err != nil {
		return ListRequest{}, err
	}
	req.Limit, err = parseInt32Query(r, "limit", req.Limit)
	if err != nil {
		return ListRequest{}, err
	}
	return req, nil
}

func parseInt32Query(r *http.Request, key string, fallback int32) (int32, error) {
	value := strings.TrimSpace(r.URL.Query().Get(key))
	if value == "" {
		return fallback, nil
	}
	parsed, err := strconv.ParseInt(value, 10, 32)
	if err != nil {
		return 0, apperror.NewBadRequest(key + " must be an integer")
	}
	return int32(parsed), nil
}

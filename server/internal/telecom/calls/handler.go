package calls

import (
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

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
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

func requestCallIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid call id")
	}
	return organizationID, id, nil
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

package messaging

import (
	"encoding/json"
	"net/http"
	"strconv"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/platform/idempotency"
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
	if !decodeRequest(w, r, &req) {
		return
	}

	value, err := h.service.Create(
		r.Context(),
		organizationID,
		r.Header.Get(idempotency.Header),
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, messageResponse(value))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := requestMessageIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	value, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, messageResponse(value))
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
	values, err := h.service.List(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items := make([]MessageResponse, 0, len(values))
	for _, value := range values {
		items = append(items, messageResponse(value))
	}
	httputil.OK(w, map[string]any{"messages": items})
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

func requestMessageIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid message id")
	}
	return organizationID, id, nil
}

func decodeRequest(w http.ResponseWriter, r *http.Request, value any) bool {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid request body"))
		return false
	}
	return true
}

func listRequest(r *http.Request) (ListRequest, error) {
	req := ListRequest{Offset: 0, Limit: 50}
	if value := strings.TrimSpace(r.URL.Query().Get("status")); value != "" {
		req.Status = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("direction")); value != "" {
		req.Direction = &value
	}
	if value := strings.TrimSpace(r.URL.Query().Get("channel")); value != "" {
		req.Channel = &value
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

package recordings

import (
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/platform/middleware"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct{ service *Service }

func NewHandler(service *Service) *Handler { return &Handler{service: service} }
func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	org, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	offset, limit, err := pagination(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	recordings, err := h.service.List(r.Context(), org, offset, limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items := make([]RecordingResponse, 0, len(recordings))
	for _, recording := range recordings {
		items = append(items, recordingResponse(recording))
	}
	httputil.OK(w, map[string]any{"recordings": items})
}
func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestRecordingIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	recording, err := h.service.Get(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, recordingResponse(recording))
}
func (h *Handler) Playback(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestRecordingIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	playback, err := h.service.Playback(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, playback)
}
func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	org, id, err := requestRecordingIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), org, id); err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]string{"message": "recording deleted"})
}
func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}
func requestRecordingIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	org, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid recording id")
	}
	return org, id, nil
}
func pagination(r *http.Request) (int32, int32, error) {
	offset, limit := int32(0), int32(50)
	if value := r.URL.Query().Get("offset"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, apperror.NewBadRequest("invalid offset")
		}
		offset = int32(parsed)
	}
	if value := r.URL.Query().Get("limit"); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, apperror.NewBadRequest("invalid limit")
		}
		limit = int32(parsed)
	}
	return offset, limit, nil
}

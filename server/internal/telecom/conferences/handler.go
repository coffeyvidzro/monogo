package conferences

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

	var req CreateRequest
	if !decode(w, r, &req) {
		return
	}

	value, err := h.service.Create(r.Context(), org, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, conferenceResponse(value))
}

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

	values, err := h.service.List(r.Context(), org, offset, limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]ConferenceResponse, 0, len(values))
	for _, value := range values {
		items = append(items, conferenceResponse(value))
	}

	httputil.OK(w, map[string]any{"conferences": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	value, err := h.service.Get(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, conferenceResponse(value))
}

func (h *Handler) End(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	value, err := h.service.End(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, conferenceResponse(value))
}

func (h *Handler) AddParticipant(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var req AddParticipantRequest
	if !decode(w, r, &req) {
		return
	}

	value, err := h.service.AddParticipant(r.Context(), org, id, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, participantResponse(value))
}

func (h *Handler) ListParticipants(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	values, err := h.service.ListParticipants(r.Context(), org, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items := make([]ParticipantResponse, 0, len(values))
	for _, value := range values {
		items = append(items, participantResponse(value))
	}

	httputil.OK(w, map[string]any{"participants": items})
}

func (h *Handler) RemoveParticipant(w http.ResponseWriter, r *http.Request) {
	h.participantAction(w, r, "left", nil, nil)
}

func (h *Handler) Mute(w http.ResponseWriter, r *http.Request) {
	value := true
	h.participantAction(w, r, "joined", &value, nil)
}

func (h *Handler) Unmute(w http.ResponseWriter, r *http.Request) {
	value := false
	h.participantAction(w, r, "joined", &value, nil)
}

func (h *Handler) Deaf(w http.ResponseWriter, r *http.Request) {
	value := true
	h.participantAction(w, r, "joined", nil, &value)
}

func (h *Handler) Undeaf(w http.ResponseWriter, r *http.Request) {
	value := false
	h.participantAction(w, r, "joined", nil, &value)
}

func (h *Handler) Kick(w http.ResponseWriter, r *http.Request) {
	h.participantAction(w, r, "failed", nil, nil)
}

func (h *Handler) participantAction(
	w http.ResponseWriter,
	r *http.Request,
	state string,
	muted *bool,
	deaf *bool,
) {
	org, conferenceID, participantID, err := participantIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	value, err := h.service.SetParticipant(
		r.Context(),
		org,
		conferenceID,
		participantID,
		state,
		muted,
		deaf,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, participantResponse(value))
}

func (h *Handler) Lock(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err == nil {
		err = h.service.Lock(r.Context(), org, id)
	}
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, map[string]string{"message": "conference lock action completed"})
}

func (h *Handler) Unlock(w http.ResponseWriter, r *http.Request) {
	org, id, err := conferenceIDs(r)
	if err == nil {
		err = h.service.Unlock(r.Context(), org, id)
	}
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, map[string]string{"message": "conference unlock action completed"})
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

func conferenceIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	org, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid conference id")
	}

	return org, id, nil
}

func participantIDs(r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	org, conferenceID, err := conferenceIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	participantID, err := uuid.Parse(chi.URLParam(r, "participant_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid conference participant id")
	}

	return org, conferenceID, participantID, nil
}

func pagination(r *http.Request) (int32, int32, error) {
	offset, limit := int32(0), int32(50)

	for _, item := range []struct {
		name  string
		value *int32
	}{
		{"offset", &offset},
		{"limit", &limit},
	} {
		if raw := r.URL.Query().Get(item.name); raw != "" {
			parsed, err := strconv.ParseInt(raw, 10, 32)
			if err != nil {
				return 0, 0, apperror.NewBadRequest("invalid " + item.name)
			}

			*item.value = int32(parsed)
		}
	}

	return offset, limit, nil
}

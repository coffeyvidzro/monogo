package lifecycle

import (
	"encoding/json"
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/helper"
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

func writeAccepted(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(value)
}

func (h *Handler) ReleaseManaged(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	op, err := h.service.ReleaseManaged(r.Context(), organizationID, numberID, r.Header.Get("Idempotency-Key"))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	w.Header().Set("Location", "/v1/numbers/lifecycle/"+op.ID.String())
	writeAccepted(w, lifecycleResponse(op))
}

func (h *Handler) GetLifecycle(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "operation_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid lifecycle operation id"))
		return
	}
	op, err := h.service.GetLifecycle(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, lifecycleResponse(op))
}

func (h *Handler) PutEmergency(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[EmergencyAddressRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	registration, err := h.service.PutEmergency(r.Context(), organizationID, numberID, r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	writeAccepted(w, emergencyResponse(registration))
}

func (h *Handler) GetEmergency(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	registration, err := h.service.GetEmergency(r.Context(), organizationID, numberID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, emergencyResponse(registration))
}

func (h *Handler) CreatePortIn(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[PortInRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	portIn, err := h.service.CreatePortIn(r.Context(), organizationID, r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	w.Header().Set("Location", "/v1/numbers/port-ins/"+portIn.Case.ID.String())
	writeAccepted(w, portInResponse(portIn))
}

func (h *Handler) GetPortIn(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "case_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid port-in case id"))
		return
	}
	portIn, err := h.service.GetPortIn(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, portInResponse(portIn))
}

func (h *Handler) AddPortDocument(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "case_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid port-in case id"))
		return
	}
	req, err := helper.DecodeJSON[PortDocumentRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	document, err := h.service.AddPortDocument(r.Context(), organizationID, id, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, portDocumentResponse(document))
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

func requestIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, "number_id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid number id")
	}
	return organizationID, id, nil
}

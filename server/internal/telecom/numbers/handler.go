package numbers

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

func (h *Handler) PurchaseNumber(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[ManagedPurchaseRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	order, err := h.service.Purchase(r.Context(), organizationID, r.Header.Get("Idempotency-Key"), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if order.Status == "completed" {
		httputil.OK(w, order)
		return
	}
	w.Header().Set("Location", "/v1/numbers/orders/"+order.ID.String())
	writeAccepted(w, order)
}

func (h *Handler) GetManagedOrder(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	id, err := uuid.Parse(chi.URLParam(r, "order_id"))
	if err != nil || id == uuid.Nil {
		httputil.Error(w, apperror.NewBadRequest("invalid order id"))
		return
	}
	order, err := h.service.GetManagedOrder(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, order)
}

func writeAccepted(w http.ResponseWriter, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusAccepted)
	_ = json.NewEncoder(w).Encode(value)
}

// SearchAvailable exposes display-only DIDWW inventory. Purchasing and
// provisioning are deliberately not implemented by this endpoint.
func (h *Handler) SearchAvailable(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	numbers, err := h.service.SearchAvailable(r.Context(), organizationID, r.URL.Query().Get("contains"))
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"numbers": numbers})
}

func (h *Handler) CreateNumber(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[CreateBYOCRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	number, err := h.service.CreateBYOC(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, response(number))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items := make([]Response, 0, len(rows))
	for _, number := range rows {
		items = append(items, response(number))
	}
	httputil.OK(w, map[string]any{"numbers": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	number, err := h.service.Get(r.Context(), organizationID, numberID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(number))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	number, err := h.service.Update(r.Context(), organizationID, numberID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(number))
}

func (h *Handler) SetBYOCConnection(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req, err := helper.DecodeJSON[SetCarrierConnectionRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	number, err := h.service.SetBYOCConnection(r.Context(), organizationID, numberID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(number))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.Delete(r.Context(), organizationID, numberID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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

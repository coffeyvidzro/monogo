package numbers

import (
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

func (h *Handler) CreateBYOC(w http.ResponseWriter, r *http.Request) {
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

func (h *Handler) ReleaseBYOC(w http.ResponseWriter, r *http.Request) {
	organizationID, numberID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	if err := h.service.ReleaseBYOC(r.Context(), organizationID, numberID); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
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
	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil || id == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid number id")
	}
	return organizationID, id, nil
}

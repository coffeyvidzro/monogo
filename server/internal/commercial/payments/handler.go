package payments

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
	return &Handler{
		service: service,
	}
}

func (h *Handler) CreateSubscription(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := paymentOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateSubscriptionRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID

	payment, err := h.service.CreateSubscription(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, payment)
}

func (h *Handler) CreateWalletTopup(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := paymentOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateWalletTopupRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID

	payment, err := h.service.CreateWalletTopup(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, payment)
}

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := paymentOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid payment id"))
		return
	}

	payment, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, payment)
}

func paymentOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

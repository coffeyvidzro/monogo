package checkout

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
	organizationID, err := checkoutOrganizationID(r)
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

	result, err := h.service.CreateSubscription(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, result)
}

func (h *Handler) CreateWalletTopup(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := checkoutOrganizationID(r)
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

	result, err := h.service.CreateWalletTopup(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, result)
}

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := checkoutOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	paymentID, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid payment id"))
		return
	}

	result, err := h.service.Get(r.Context(), organizationID, paymentID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, result)
}

func checkoutOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

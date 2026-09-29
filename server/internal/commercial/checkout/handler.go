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

func (h *Handler) Create(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := checkoutOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID

	result, err := h.service.Create(r.Context(), req)
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

	checkoutID, err := uuid.Parse(chi.URLParam(r, "checkout_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid checkout id"))
		return
	}

	result, err := h.service.Get(r.Context(), organizationID, checkoutID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, result)
}

func (h *Handler) Confirm(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, checkoutID, err := checkoutRequestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[ConfirmRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID
	req.CheckoutID = checkoutID

	result, err := h.service.Confirm(r.Context(), req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, result)
}

func (h *Handler) Continue(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, checkoutID, err := checkoutRequestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[ContinueRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID
	req.CheckoutID = checkoutID

	result, err := h.service.Continue(
		r.Context(),
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, result)
}

func checkoutRequestIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := checkoutOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	checkoutID, err := uuid.Parse(chi.URLParam(r, "checkout_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid checkout id")
	}

	return organizationID, checkoutID, nil
}

func checkoutOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

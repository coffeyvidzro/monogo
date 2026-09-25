package payments

import (
	"context"
	"net/http"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/helper"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type paymentService interface {
	Initiate(context.Context, uuid.UUID, uuid.UUID, string, CreateInput) (InitiationResult, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.Payment, error)
}

type Handler struct{ service paymentService }

func NewHandler(service paymentService) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, err := paymentOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	checkoutID, err := uuid.Parse(chi.URLParam(r, "checkout_id"))
	if err != nil || checkoutID == uuid.Nil {
		httputil.Error(w, apperror.NewBadRequest("invalid checkout id"))
		return
	}
	input, err := helper.DecodeJSON[CreateInput](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	result, err := h.service.Initiate(r.Context(), organizationID, checkoutID, r.Header.Get("Idempotency-Key"), input)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	w.Header().Set("Location", "/v1/payments/"+result.Payment.ID.String())
	httputil.Created(w, InitiationResponse{
		Response:       response(result.Payment),
		ClientSecret:   result.ClientSecret,
		DisplayText:    result.DisplayText,
		ProviderStatus: result.ProviderStatus,
	})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, err := paymentOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	paymentID, err := uuid.Parse(chi.URLParam(r, "payment_id"))
	if err != nil || paymentID == uuid.Nil {
		httputil.Error(w, apperror.NewBadRequest("invalid payment id"))
		return
	}
	row, err := h.service.Get(r.Context(), organizationID, paymentID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(row))
}

func paymentOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

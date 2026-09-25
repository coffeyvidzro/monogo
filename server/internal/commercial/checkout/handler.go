package checkout

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

type checkoutService interface {
	Create(context.Context, CreateRequest) (sqlc.Checkout, error)
	Get(context.Context, uuid.UUID, uuid.UUID) (sqlc.Checkout, error)
	Cancel(context.Context, uuid.UUID, uuid.UUID) (sqlc.Checkout, error)
}

type Handler struct{ service checkoutService }

func NewHandler(service checkoutService) *Handler { return &Handler{service: service} }

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, err := checkoutOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	input, err := helper.DecodeJSON[CreateInput](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	row, err := h.service.Create(r.Context(), CreateRequest{OrganizationID: organizationID,
		WalletID: input.WalletID, AmountMinor: input.AmountMinor, Currency: input.Currency,
		IdempotencyKey: r.Header.Get("Idempotency-Key"), ExpiresAt: input.ExpiresAt})
	if err != nil {
		httputil.Error(w, err)
		return
	}
	w.Header().Set("Location", "/v1/checkouts/"+row.ID.String())
	httputil.Created(w, response(row))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, checkoutID, err := checkoutRequestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	row, err := h.service.Get(r.Context(), organizationID, checkoutID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(row))
}

func (h *Handler) Cancel(w http.ResponseWriter, r *http.Request) {
	organizationID, checkoutID, err := checkoutRequestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	row, err := h.service.Cancel(r.Context(), organizationID, checkoutID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(row))
}

func checkoutOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}
	return id, nil
}

func checkoutRequestIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := checkoutOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	checkoutID, err := uuid.Parse(chi.URLParam(r, "checkout_id"))
	if err != nil || checkoutID == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid checkout id")
	}
	return organizationID, checkoutID, nil
}

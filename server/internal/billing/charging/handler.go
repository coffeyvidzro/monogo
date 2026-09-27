package charging

import (
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	if service == nil {
		panic("billing charging: service is required")
	}

	return &Handler{
		service: service,
	}
}

func (h *Handler) Reserve(w http.ResponseWriter, r *http.Request) {
	h.chargeMutation(w, r, h.service.Reserve)
}

func (h *Handler) Consume(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, h.service.Consume)
}

func (h *Handler) Release(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, h.service.Release)
}

func (h *Handler) Debit(w http.ResponseWriter, r *http.Request) {
	h.mutate(w, r, h.service.Debit)
}

func (h *Handler) Finalize(w http.ResponseWriter, r *http.Request) {
	organizationID, resourceID, err := requestIDs(r, "charge_id")
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var body FinalizeOperationRequest
	if err := decodeRequest(r, &body); err != nil {
		httputil.Error(w, err)
		return
	}
	defaultOccurredAt(&body.OccurredAt)

	result, err := h.service.Finalize(
		r.Context(),
		FinalizeRequest{
			OrganizationID: organizationID,
			ChargeID:       resourceID,
			OperationID:    body.OperationID,
			Status:         body.Status,
			OccurredAt:     body.OccurredAt,
		},
	)
	writeResult(w, result, err)
}

func (h *Handler) Credit(w http.ResponseWriter, r *http.Request) {
	organizationID, resourceID, err := requestIDs(r, "wallet_id")
	if err != nil {
		httputil.Error(w, err)
		return
	}

	var body CreditOperationRequest
	if err := decodeRequest(r, &body); err != nil {
		httputil.Error(w, err)
		return
	}
	defaultOccurredAt(&body.OccurredAt)

	result, err := h.service.Credit(
		r.Context(),
		CreditRequest{
			OrganizationID: organizationID,
			WalletID:       resourceID,
			OperationID:    body.OperationID,
			AmountMicros:   body.AmountMicros,
			OccurredAt:     body.OccurredAt,
		},
	)
	writeResult(w, result, err)
}

func (h *Handler) chargeMutation(
	w http.ResponseWriter,
	r *http.Request,
	operation func(context.Context, ReserveRequest) (Result, error),
) {
	organizationID, chargeID, body, ok := mutationRequest(w, r)
	if !ok {
		return
	}

	result, err := operation(
		r.Context(),
		ReserveRequest{
			OrganizationID: organizationID,
			ChargeID:       chargeID,
			OperationID:    body.OperationID,
			AmountMicros:   body.AmountMicros,
			OccurredAt:     body.OccurredAt,
		},
	)
	writeResult(w, result, err)
}

func (h *Handler) mutate(
	w http.ResponseWriter,
	r *http.Request,
	operation func(context.Context, MutationRequest) (Result, error),
) {
	organizationID, chargeID, body, ok := mutationRequest(w, r)
	if !ok {
		return
	}

	result, err := operation(
		r.Context(),
		MutationRequest{
			OrganizationID: organizationID,
			ChargeID:       chargeID,
			OperationID:    body.OperationID,
			AmountMicros:   body.AmountMicros,
			OccurredAt:     body.OccurredAt,
		},
	)
	writeResult(w, result, err)
}

func mutationRequest(
	w http.ResponseWriter,
	r *http.Request,
) (uuid.UUID, uuid.UUID, OperationRequest, bool) {
	organizationID, chargeID, err := requestIDs(r, "charge_id")
	if err != nil {
		httputil.Error(w, err)
		return uuid.Nil, uuid.Nil, OperationRequest{}, false
	}

	var body OperationRequest
	if err := decodeRequest(r, &body); err != nil {
		httputil.Error(w, err)
		return uuid.Nil, uuid.Nil, OperationRequest{}, false
	}
	defaultOccurredAt(&body.OccurredAt)

	return organizationID, chargeID, body, true
}

func requestIDs(r *http.Request, parameter string) (uuid.UUID, uuid.UUID, error) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest(
			"organization context required",
		)
	}

	resourceID, err := uuid.Parse(chi.URLParam(r, parameter))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest(
			"invalid " + parameter,
		)
	}

	return organizationID, resourceID, nil
}

func decodeRequest(r *http.Request, value any) error {
	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(value); err != nil {
		return apperror.NewBadRequest("invalid request body")
	}
	if err := decoder.Decode(&struct{}{}); !errors.Is(err, io.EOF) {
		return apperror.NewBadRequest("request body must contain one JSON object")
	}

	return nil
}

func defaultOccurredAt(value *time.Time) {
	if value.IsZero() {
		*value = time.Now().UTC()
	}
}

func writeResult(w http.ResponseWriter, result Result, err error) {
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, response(result))
}

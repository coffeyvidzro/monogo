package wallets

import (
	"errors"
	"net/http"
	"strconv"
	"strings"

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
	return &Handler{
		service: service,
	}
}

func (h *Handler) GetByCurrency(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.GetByCurrency(
		r.Context(),
		organizationID,
		chi.URLParam(r, "currency"),
	)
	if err != nil {
		httputil.Error(w, walletHTTPError(err))
		return
	}

	httputil.OK(w, item)
}

func (h *Handler) ListLedger(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	wallet, err := h.service.GetByCurrency(
		r.Context(),
		organizationID,
		chi.URLParam(r, "currency"),
	)
	if err != nil {
		httputil.Error(w, walletHTTPError(err))
		return
	}

	limit, offset, err := ledgerPagination(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	entries, err := h.service.ListLedgerEntries(
		r.Context(),
		ListLedgerRequest{
			OrganizationID: organizationID,
			WalletID:       wallet.ID,
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		httputil.Error(w, walletHTTPError(err))
		return
	}

	httputil.OK(
		w,
		map[string]any{
			"wallet":         wallet,
			"ledger_entries": entries,
		},
	)
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

func ledgerPagination(r *http.Request) (int32, int32, error) {
	limit := int32(50)
	offset := int32(0)

	if value := strings.TrimSpace(r.URL.Query().Get("limit")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, apperror.NewBadRequest("invalid limit")
		}
		limit = int32(parsed)
	}

	if value := strings.TrimSpace(r.URL.Query().Get("offset")); value != "" {
		parsed, err := strconv.ParseInt(value, 10, 32)
		if err != nil {
			return 0, 0, apperror.NewBadRequest("invalid offset")
		}
		offset = int32(parsed)
	}

	return limit, offset, nil
}

func walletHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return apperror.NewBadRequest(err.Error())
	case errors.Is(err, ErrNotFound):
		return apperror.NewNotFound("wallet not found")
	case errors.Is(err, ErrInsufficientBalance):
		return apperror.NewPaymentRequired("insufficient wallet balance")
	case errors.Is(err, ErrInvalidState):
		return apperror.NewConflict("wallet state does not allow operation")
	case errors.Is(err, ErrOperationConflict):
		return apperror.NewConflict("wallet operation conflicts with existing ledger entry")
	default:
		return apperror.NewInternal("commercial wallet operation failed", err)
	}
}

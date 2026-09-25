package wallets

import (
	"context"
	"net/http"
	"strconv"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

type reader interface {
	List(context.Context, uuid.UUID) ([]sqlc.Wallet, error)
	GetByID(context.Context, uuid.UUID, uuid.UUID) (sqlc.Wallet, error)
	ListTransactions(context.Context, uuid.UUID, uuid.UUID, int32) ([]sqlc.WalletTransaction, error)
}

type Handler struct{ service reader }

func NewHandler(service reader) *Handler { return &Handler{service: service} }

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
	for _, row := range rows {
		items = append(items, response(row))
	}
	httputil.OK(w, map[string]any{"wallets": items})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, walletID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	row, err := h.service.GetByID(r.Context(), organizationID, walletID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(row))
}

func (h *Handler) ListTransactions(w http.ResponseWriter, r *http.Request) {
	organizationID, walletID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	limit, err := transactionLimit(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	rows, err := h.service.ListTransactions(r.Context(), organizationID, walletID, limit)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	items := make([]TransactionResponse, 0, len(rows))
	for _, row := range rows {
		items = append(items, transactionResponse(row))
	}
	httputil.OK(w, map[string]any{"transactions": items})
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
	walletID, err := uuid.Parse(chi.URLParam(r, "wallet_id"))
	if err != nil || walletID == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid wallet id")
	}
	return organizationID, walletID, nil
}

func transactionLimit(r *http.Request) (int32, error) {
	raw := r.URL.Query().Get("limit")
	if raw == "" {
		return defaultTransactionLimit, nil
	}
	value, err := strconv.ParseInt(raw, 10, 32)
	if err != nil || value < 1 || value > int64(maxTransactionLimit) {
		return 0, apperror.NewBadRequest("transaction limit must be between 1 and 200")
	}
	return int32(value), nil
}

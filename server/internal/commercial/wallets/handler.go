package wallets

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/platform/middleware"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/httputil"
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

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Get(
		r.Context(),
		organizationID,
	)
	if err != nil {
		httputil.Error(w, err)
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

	if _, err := h.service.Get(
		r.Context(),
		organizationID,
	); err != nil {
		httputil.Error(w, err)
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
			Limit:          limit,
			Offset:         offset,
		},
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(
		w,
		map[string]any{
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

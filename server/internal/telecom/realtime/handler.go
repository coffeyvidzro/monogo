package realtime

import (
	"errors"
	"net/http"

	"github.com/leamout/leamout/internal/platform/middleware"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	if service == nil {
		panic("realtime: service is required")
	}
	return &Handler{service: service}
}

func (h *Handler) IssueICECredentials(w http.ResponseWriter, r *http.Request) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
		return
	}

	credentials, err := h.service.Issue(r.Context(), organizationID)
	if err != nil {
		if errors.Is(err, ErrIssueRateLimited) {
			httputil.Error(w, apperror.NewTooManyRequests("TURN credential issuance rate limit exceeded"))
			return
		}
		httputil.Error(w, apperror.NewInternal("issue TURN credentials", err))
		return
	}
	w.Header().Set("Cache-Control", "no-store")
	w.Header().Set("Pragma", "no-cache")
	httputil.OK(w, credentials)
}

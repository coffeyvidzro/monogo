package pricing

import (
	"errors"
	"net/http"
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

func (h *Handler) Resolve(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || organizationID == uuid.Nil {
		httputil.Error(w, apperror.NewBadRequest("organization context required"))
		return
	}

	item, err := h.service.Resolve(
		r.Context(),
		ResolveRequest{
			OrganizationID:    organizationID,
			DestinationDigits: strings.TrimSpace(r.URL.Query().Get("destination")),
			Direction:         strings.TrimSpace(r.URL.Query().Get("direction")),
			Currency:          strings.TrimSpace(r.URL.Query().Get("currency")),
		},
	)
	if err != nil {
		httputil.Error(w, pricingHTTPError(err))
		return
	}

	httputil.OK(w, item)
}

func pricingHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return apperror.NewBadRequest(err.Error())
	case errors.Is(err, ErrRateNotFound):
		return apperror.NewNotFound("carrier rate not found")
	case errors.Is(err, ErrRateConflict):
		return apperror.NewConflict("carrier rate conflict")
	default:
		return apperror.NewInternal("commercial pricing operation failed", err)
	}
}

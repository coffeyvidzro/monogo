package subscriptions

import (
	"errors"
	"net/http"

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

func (h *Handler) ListPlans(
	w http.ResponseWriter,
	r *http.Request,
) {
	plans, err := h.service.ListPlans(r.Context())
	if err != nil {
		httputil.Error(w, subscriptionHTTPError(err))
		return
	}

	httputil.OK(
		w,
		map[string]any{
			"subscription_plans": plans,
		},
	)
}

func (h *Handler) GetPlan(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(chi.URLParam(r, "plan_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid subscription plan id"))
		return
	}

	plan, err := h.service.GetPlan(
		r.Context(),
		id,
	)
	if err != nil {
		httputil.Error(w, subscriptionHTTPError(err))
		return
	}

	httputil.OK(w, plan)
}

func (h *Handler) Current(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	subscription, err := h.service.Current(
		r.Context(),
		organizationID,
	)
	if err != nil {
		httputil.Error(w, subscriptionHTTPError(err))
		return
	}

	httputil.OK(w, subscription)
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

func subscriptionHTTPError(err error) error {
	switch {
	case errors.Is(err, ErrInvalidInput):
		return apperror.NewBadRequest(err.Error())
	case errors.Is(err, ErrPlanNotFound):
		return apperror.NewNotFound("subscription plan not found")
	case errors.Is(err, ErrSubscriptionNotFound):
		return apperror.NewNotFound("subscription not found")
	case errors.Is(err, ErrPlanConflict),
		errors.Is(err, ErrSubscriptionConflict),
		errors.Is(err, ErrSubscriptionInvalidState):
		return apperror.NewConflict(err.Error())
	case errors.Is(err, ErrSubscriptionNotPermitted):
		return apperror.NewForbidden("subscription is not permitted")
	default:
		return apperror.NewInternal("commercial subscription operation failed", err)
	}
}

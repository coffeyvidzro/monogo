package subscriptions

import (
	"net/http"
	"strconv"
	"strings"

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
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[SubscribeRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	req.OrganizationID = organizationID

	subscription, err := h.service.Subscribe(
		r.Context(),
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, subscription)
}

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, id, err := requestSubscriptionIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	subscription, err := h.service.Get(
		r.Context(),
		organizationID,
		id,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, subscription)
}

func (h *Handler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	limit, offset, err := subscriptionPagination(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.List(
		r.Context(),
		ListRequest{
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
			"subscriptions": items,
		},
	)
}

func (h *Handler) Update(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, id, err := requestSubscriptionIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	subscription, err := h.service.Update(
		r.Context(),
		organizationID,
		id,
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, subscription)
}

func (h *Handler) Delete(
	w http.ResponseWriter,
	r *http.Request,
) {
	organizationID, id, err := requestSubscriptionIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if _, err := h.service.Cancel(
		r.Context(),
		organizationID,
		id,
	); err != nil {
		httputil.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requestOrganizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest("organization context required")
	}

	return id, nil
}

func requestSubscriptionIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := requestOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	id, err := uuid.Parse(chi.URLParam(r, "id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid subscription id")
	}

	return organizationID, id, nil
}

func subscriptionPagination(r *http.Request) (int32, int32, error) {
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

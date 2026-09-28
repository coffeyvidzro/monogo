package plans

import (
	"net/http"

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

func (h *Handler) List(
	w http.ResponseWriter,
	r *http.Request,
) {
	items, err := h.service.List(r.Context())
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(
		w,
		map[string]any{
			"plans": items,
		},
	)
}

func (h *Handler) Get(
	w http.ResponseWriter,
	r *http.Request,
) {
	id, err := uuid.Parse(chi.URLParam(r, "plan_id"))
	if err != nil {
		httputil.Error(w, apperror.NewBadRequest("invalid plan id"))
		return
	}

	item, err := h.service.Get(
		r.Context(),
		id,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, item)
}

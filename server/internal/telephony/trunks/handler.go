package trunks

import (
	"net/http"

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
	return &Handler{service: service}
}

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Create(r.Context(), organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, response(item))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, err := organizationID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	result := make([]Response, 0, len(items))
	for _, item := range items {
		result = append(result, response(item))
	}
	httputil.OK(w, map[string]any{"trunks": result})
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Get(r.Context(), organizationID, trunkID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(item))
}

func (h *Handler) Validate(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	result, err := h.service.Validate(r.Context(), organizationID, trunkID)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, result)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Update(
		r.Context(),
		organizationID,
		trunkID,
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, response(item))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Delete(
		r.Context(),
		organizationID,
		trunkID,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetOutboundAuth(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[AuthRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.SetOutboundAuth(
		r.Context(),
		organizationID,
		trunkID,
		req,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ClearOutboundAuth(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.ClearOutboundAuth(
		r.Context(),
		organizationID,
		trunkID,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) SetInboundAuth(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[AuthRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.SetInboundAuth(
		r.Context(),
		organizationID,
		trunkID,
		req,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) ClearInboundAuth(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.ClearInboundAuth(
		r.Context(),
		organizationID,
		trunkID,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateSourceIP(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[SourceIPRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.CreateSourceIP(
		r.Context(),
		organizationID,
		trunkID,
		req.CIDR,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, item)
}

func (h *Handler) ListSourceIPs(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.ListSourceIPs(
		r.Context(),
		organizationID,
		trunkID,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, map[string]any{"source_ips": items})
}

func (h *Handler) DeleteSourceIP(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, sourceIPID, err := sourceIPIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.DeleteSourceIP(
		r.Context(),
		organizationID,
		trunkID,
		sourceIPID,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateEndpoint(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[EndpointCreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.CreateEndpoint(
		r.Context(),
		organizationID,
		trunkID,
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.Created(w, endpointResponse(item))
}

func (h *Handler) ListEndpoints(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.ListEndpoints(
		r.Context(),
		organizationID,
		trunkID,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	result := make([]EndpointResponse, 0, len(items))
	for _, item := range items {
		result = append(result, endpointResponse(item))
	}
	httputil.OK(w, map[string]any{"endpoints": result})
}

func (h *Handler) GetEndpoint(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, endpointID, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.GetEndpoint(
		r.Context(),
		organizationID,
		trunkID,
		endpointID,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, endpointResponse(item))
}

func (h *Handler) UpdateEndpoint(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, endpointID, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[EndpointUpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.UpdateEndpoint(
		r.Context(),
		organizationID,
		trunkID,
		endpointID,
		req,
	)
	if err != nil {
		httputil.Error(w, err)
		return
	}
	httputil.OK(w, endpointResponse(item))
}

func (h *Handler) DeleteEndpoint(w http.ResponseWriter, r *http.Request) {
	organizationID, trunkID, endpointID, err := endpointIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.DeleteEndpoint(
		r.Context(),
		organizationID,
		trunkID,
		endpointID,
	); err != nil {
		httputil.Error(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func organizationID(r *http.Request) (uuid.UUID, error) {
	id, ok := middleware.OrganizationIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewBadRequest(
			"organization context required",
		)
	}
	return id, nil
}

func trunkIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := organizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	trunkID, err := uuid.Parse(chi.URLParam(r, "trunk_id"))
	if err != nil || trunkID == uuid.Nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest(
			"invalid trunk_id",
		)
	}
	return organizationID, trunkID, nil
}

func sourceIPIDs(
	r *http.Request,
) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	sourceIPID, err := uuid.Parse(chi.URLParam(r, "source_ip_id"))
	if err != nil || sourceIPID == uuid.Nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest(
			"invalid source_ip_id",
		)
	}
	return organizationID, trunkID, sourceIPID, nil
}

func endpointIDs(
	r *http.Request,
) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organizationID, trunkID, err := trunkIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	endpointID, err := uuid.Parse(chi.URLParam(r, "endpoint_id"))
	if err != nil || endpointID == uuid.Nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest(
			"invalid endpoint_id",
		)
	}
	return organizationID, trunkID, endpointID, nil
}

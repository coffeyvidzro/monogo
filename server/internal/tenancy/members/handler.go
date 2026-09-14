package members

import (
	"net/http"

	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/security/authn"
	"github.com/leamout/leamout/pkg/apperror"
	"github.com/leamout/leamout/pkg/helper"
	"github.com/leamout/leamout/pkg/httputil"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) Add(w http.ResponseWriter, r *http.Request) {
	requesterID, organizationID, err := requestOrganizationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	member, err := h.service.Add(r.Context(), requesterID, organizationID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, toResponse(member))
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	requesterID, organizationID, err := requestOrganizationIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	members, err := h.service.List(r.Context(), requesterID, organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, toResponses(members))
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	requesterID, organizationID, memberID, err := requestMemberIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	member, err := h.service.Get(r.Context(), requesterID, organizationID, memberID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, toResponse(member))
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	requesterID, organizationID, memberID, err := requestMemberIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	member, err := h.service.Update(r.Context(), requesterID, organizationID, memberID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, toResponse(member))
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	requesterID, organizationID, memberID, err := requestMemberIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), requesterID, organizationID, memberID); err != nil {
		httputil.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requestOrganizationIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	requesterID, err := authenticatedUserID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	organizationID, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid organization_id")
	}

	return requesterID, organizationID, nil
}

func requestMemberIDs(r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	requesterID, organizationID, err := requestOrganizationIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	memberID, err := uuid.Parse(chi.URLParam(r, "user_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid user_id")
	}

	return requesterID, organizationID, memberID, nil
}

func authenticatedUserID(r *http.Request) (uuid.UUID, error) {
	userID, ok := authn.UserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorized("authentication required")
	}

	return userID, nil
}

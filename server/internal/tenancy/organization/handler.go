package organization

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

func (h *Handler) Create(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[CreateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	org, err := h.service.Create(r.Context(), userID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, org)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	userID, orgID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	org, err := h.service.Get(r.Context(), userID, orgID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, org)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	userID, orgID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[UpdateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	org, err := h.service.Update(r.Context(), userID, orgID, req)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, org)
}

func (h *Handler) Delete(w http.ResponseWriter, r *http.Request) {
	userID, orgID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), userID, orgID); err != nil {
		httputil.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	orgs, err := h.service.List(r.Context(), userID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, orgs)
}

func requestIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	orgID, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid organization_id")
	}

	return userID, orgID, nil
}

func authenticatedUserID(r *http.Request) (uuid.UUID, error) {
	userID, ok := authn.UserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorized("authentication required")
	}

	return userID, nil
}

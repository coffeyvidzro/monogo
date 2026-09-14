package users

import (
	"net/http"

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

func (h *Handler) GetMe(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	user, err := h.service.Get(r.Context(), userID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, toResponse(user))
}

func (h *Handler) UpdateMe(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	req, err := helper.DecodeJSON[UpdateProfileRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := validateUpdateProfileRequest(req); err != nil {
		httputil.Error(w, err)
		return
	}

	user, err := h.service.UpdateProfile(r.Context(), userID, req.Name)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, toResponse(user))
}

func (h *Handler) DeleteMe(w http.ResponseWriter, r *http.Request) {
	userID, err := authenticatedUserID(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Delete(r.Context(), userID); err != nil {
		httputil.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func authenticatedUserID(r *http.Request) (uuid.UUID, error) {
	userID, ok := authn.UserIDFromContext(r.Context())
	if !ok || userID == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorized("authentication required")
	}

	return userID, nil
}

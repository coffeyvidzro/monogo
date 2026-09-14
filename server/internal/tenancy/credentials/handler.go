package credentials

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
	organizationID, actorID, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	request, err := helper.DecodeJSON[createRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	created, err := h.service.Create(r.Context(), CreateInput{
		OrganizationID: organizationID,
		CreatedBy:      actorID,
		Name:           request.Name,
		Description:    request.Description,
		Scopes:         request.Scopes,
		ExpiresAt:      request.ExpiresAt,
	})
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.Created(w, created)
}

func (h *Handler) List(w http.ResponseWriter, r *http.Request) {
	organizationID, _, err := requestIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	items, err := h.service.List(r.Context(), organizationID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, items)
}

func (h *Handler) Get(w http.ResponseWriter, r *http.Request) {
	organizationID, _, id, err := requestCredentialIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Get(r.Context(), organizationID, id)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, item)
}

func (h *Handler) Update(w http.ResponseWriter, r *http.Request) {
	organizationID, actorID, id, err := requestCredentialIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	request, err := helper.DecodeJSON[updateRequest](r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	item, err := h.service.Update(r.Context(), UpdateInput{
		ID:             id,
		OrganizationID: organizationID,
		Name:           request.Name,
		Description:    request.Description,
		Scopes:         request.Scopes,
		ExpiresAt:      request.ExpiresAt,
	}, actorID)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	httputil.OK(w, item)
}

func (h *Handler) Disable(w http.ResponseWriter, r *http.Request) {
	organizationID, actorID, id, err := requestCredentialIDs(r)
	if err != nil {
		httputil.Error(w, err)
		return
	}

	if err := h.service.Disable(r.Context(), organizationID, id, actorID); err != nil {
		httputil.Error(w, err)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func requestIDs(r *http.Request) (uuid.UUID, uuid.UUID, error) {
	actorID, err := authenticatedUserID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}

	organizationID, err := uuid.Parse(chi.URLParam(r, "organization_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid organization_id")
	}

	return organizationID, actorID, nil
}

func requestCredentialIDs(r *http.Request) (uuid.UUID, uuid.UUID, uuid.UUID, error) {
	organizationID, actorID, err := requestIDs(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, err
	}

	id, err := uuid.Parse(chi.URLParam(r, "credential_id"))
	if err != nil {
		return uuid.Nil, uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid credential_id")
	}

	return organizationID, actorID, id, nil
}

func authenticatedUserID(r *http.Request) (uuid.UUID, error) {
	id, ok := authn.UserIDFromContext(r.Context())
	if !ok || id == uuid.Nil {
		return uuid.Nil, apperror.NewUnauthorized("authentication required")
	}

	return id, nil
}

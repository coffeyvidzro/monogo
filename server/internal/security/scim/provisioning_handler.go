package scim

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/coffeyvidzro/monogo/pkg/helper"
	"github.com/go-chi/chi/v5"
	"github.com/google/uuid"
)

func (h *Handler) CreateUser(w http.ResponseWriter, r *http.Request) {
	organizationID, err := scimOrganizationID(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	input, err := helper.DecodeJSON[UserInput](r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.CreateUser(r.Context(), organizationID, input)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusCreated, value)
}

func (h *Handler) ListUsers(w http.ResponseWriter, r *http.Request) {
	organizationID, err := scimOrganizationID(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	startIndex, count, err := pagination(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.ListUsers(r.Context(), organizationID, startIndex, count, r.URL.Query().Get("filter"))
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) GetUser(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "user_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.GetUser(r.Context(), organizationID, id)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) ReplaceUser(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "user_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	input, err := helper.DecodeJSON[UserInput](r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.ReplaceUser(r.Context(), organizationID, id, input)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) DeleteUser(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "user_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	if err := h.service.DeleteUser(r.Context(), organizationID, id); err != nil {
		writeSCIMError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func (h *Handler) CreateGroup(w http.ResponseWriter, r *http.Request) {
	organizationID, err := scimOrganizationID(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	input, err := helper.DecodeJSON[GroupInput](r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.CreateGroup(r.Context(), organizationID, input)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusCreated, value)
}

func (h *Handler) ListGroups(w http.ResponseWriter, r *http.Request) {
	organizationID, err := scimOrganizationID(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	startIndex, count, err := pagination(r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.ListGroups(r.Context(), organizationID, startIndex, count, r.URL.Query().Get("filter"))
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) GetGroup(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "group_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.GetGroup(r.Context(), organizationID, id)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) ReplaceGroup(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "group_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	input, err := helper.DecodeJSON[GroupInput](r)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	value, err := h.service.ReplaceGroup(r.Context(), organizationID, id, input)
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	writeSCIM(w, http.StatusOK, value)
}

func (h *Handler) DeleteGroup(w http.ResponseWriter, r *http.Request) {
	organizationID, id, err := scimResourceIDs(r, "group_id")
	if err != nil {
		writeSCIMError(w, err)
		return
	}
	if err := h.service.DeleteGroup(r.Context(), organizationID, id); err != nil {
		writeSCIMError(w, err)
		return
	}
	w.WriteHeader(http.StatusNoContent)
}

func scimOrganizationID(r *http.Request) (uuid.UUID, error) {
	principal, ok := PrincipalFromContext(r.Context())
	if !ok {
		return uuid.Nil, apperror.NewUnauthorized("SCIM bearer token required")
	}
	return principal.OrganizationID, nil
}

func scimResourceIDs(r *http.Request, param string) (uuid.UUID, uuid.UUID, error) {
	organizationID, err := scimOrganizationID(r)
	if err != nil {
		return uuid.Nil, uuid.Nil, err
	}
	id, err := uuid.Parse(chi.URLParam(r, param))
	if err != nil {
		return uuid.Nil, uuid.Nil, apperror.NewBadRequest("invalid SCIM resource id")
	}
	return organizationID, id, nil
}

func pagination(r *http.Request) (int, int, error) {
	startIndex := 1
	count := defaultPageCount
	var err error
	if value := r.URL.Query().Get("startIndex"); value != "" {
		startIndex, err = strconv.Atoi(value)
		if err != nil || startIndex < 1 {
			return 0, 0, apperror.NewBadRequest("SCIM startIndex must be a positive integer")
		}
	}
	if value := r.URL.Query().Get("count"); value != "" {
		count, err = strconv.Atoi(value)
		if err != nil || count < 0 {
			return 0, 0, apperror.NewBadRequest("SCIM count must be a non-negative integer")
		}
	}
	return startIndex, count, nil
}

func writeSCIM(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/scim+json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeSCIMError(w http.ResponseWriter, err error) {
	status := http.StatusInternalServerError
	detail := "internal server error"
	var appErr *apperror.AppError
	if errors.As(err, &appErr) {
		status = appErr.Status
		detail = appErr.Message
	}
	writeSCIM(w, status, ErrorResponse{
		Schemas: []string{errorSchema},
		Detail:  detail,
		Status:  strconv.Itoa(status),
	})
}

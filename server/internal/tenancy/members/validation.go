package members

import (
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

const (
	roleOwner  = "owner"
	roleAdmin  = "admin"
	roleMember = "member"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}

	return nil
}

func validateUserID(id uuid.UUID, field string) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest(field + " is required")
	}

	return nil
}

func normalizeRole(role string) (string, error) {
	normalized := strings.ToLower(strings.TrimSpace(role))
	if normalized == "" {
		normalized = roleMember
	}

	if normalized != roleOwner && normalized != roleAdmin && normalized != roleMember {
		return "", apperror.NewBadRequest("role must be owner, admin, or member")
	}

	return normalized, nil
}

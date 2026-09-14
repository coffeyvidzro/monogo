package organization

import (
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}

	return nil
}

func normalizeRequiredString(value string, field string) (string, error) {
	normalized := strings.TrimSpace(value)
	if normalized == "" {
		return "", apperror.NewBadRequest(field + " is required")
	}

	return normalized, nil
}

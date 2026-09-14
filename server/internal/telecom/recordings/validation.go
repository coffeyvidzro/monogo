package recordings

import (
	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func validateOrganizationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	return nil
}
func validatePagination(offset, limit int32) error {
	if offset < 0 {
		return apperror.NewBadRequest("offset cannot be negative")
	}
	if limit < 1 || limit > 100 {
		return apperror.NewBadRequest("limit must be between 1 and 100")
	}
	return nil
}

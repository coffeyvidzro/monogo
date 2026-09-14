package subscribers

import (
	"strings"
	"unicode"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func validOrg(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization_id is required")
	}
	return nil
}
func validID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("subscriber id is required")
	}
	return nil
}
func normalizeUsername(v string) (string, error) {
	v = strings.TrimSpace(v)
	if len(v) < 1 || len(v) > 64 {
		return "", apperror.NewBadRequest("username must be between 1 and 64 characters")
	}
	for _, r := range v {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) && !strings.ContainsRune("._-", r) {
			return "", apperror.NewBadRequest("username contains invalid characters")
		}
	}
	return v, nil
}
func validatePassword(v string) error {
	if len(v) < 12 || len(v) > 1024 {
		return apperror.NewBadRequest("password must be between 12 and 1024 characters")
	}
	return nil
}
func normalizeDisplayName(v *string) (*string, error) {
	if v == nil {
		return nil, nil
	}
	s := strings.TrimSpace(*v)
	if s == "" || len(s) > 255 {
		return nil, apperror.NewBadRequest("display_name must be between 1 and 255 characters")
	}
	return &s, nil
}

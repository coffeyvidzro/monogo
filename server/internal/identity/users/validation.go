package users

import (
	"strings"

	"github.com/leamout/leamout/pkg/apperror"
)

func validateUpdateProfileRequest(req UpdateProfileRequest) error {
	if req.Name == nil {
		return apperror.NewBadRequest("name is required")
	}

	if strings.TrimSpace(*req.Name) == "" {
		return apperror.NewBadRequest("name cannot be empty")
	}

	if len([]rune(strings.TrimSpace(*req.Name))) > 200 {
		return apperror.NewBadRequest("name is too long")
	}

	return nil
}

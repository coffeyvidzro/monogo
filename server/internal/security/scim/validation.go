package scim

import (
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func normalizeCreateToken(req CreateTokenRequest, now time.Time) (CreateTokenRequest, error) {
	req.Name = strings.TrimSpace(req.Name)
	if req.Name == "" || len(req.Name) > 128 {
		return CreateTokenRequest{}, apperror.NewBadRequest("name must be between 1 and 128 characters")
	}
	if req.ExpiresAt != nil && !req.ExpiresAt.After(now) {
		return CreateTokenRequest{}, apperror.NewBadRequest("expires_at must be in the future")
	}
	return req, nil
}

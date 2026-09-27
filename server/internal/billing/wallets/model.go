package wallets

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

const (
	StatusActive = "active"
	StatusFrozen = "frozen"
	StatusClosed = "closed"
)

func normalizeCurrency(value string) (string, error) {
	value = strings.ToUpper(strings.TrimSpace(value))
	if len(value) != 3 {
		return "", apperror.NewBadRequest("currency must be a 3-letter ISO code")
	}
	for _, r := range value {
		if r < 'A' || r > 'Z' {
			return "", apperror.NewBadRequest("currency must be a 3-letter ISO code")
		}
	}
	return value, nil
}

func validStatus(value string) bool {
	switch value {
	case StatusActive, StatusFrozen, StatusClosed:
		return true
	default:
		return false
	}
}

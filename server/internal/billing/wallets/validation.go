package wallets

import (
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
)

func normalizeCurrency(value string) (string, error) {
	value = strings.ToUpper(
		strings.TrimSpace(value),
	)
	if len(value) != 3 {
		return "", apperror.NewBadRequest(
			"currency must be a 3-letter ISO code",
		)
	}

	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return "", apperror.NewBadRequest(
				"currency must be a 3-letter ISO code",
			)
		}
	}

	return value, nil
}

func normalizeStatus(value string) (string, error) {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	switch value {
	case StatusActive:
		return value, nil
	case StatusFrozen:
		return value, nil
	case StatusClosed:
		return value, nil
	default:
		return "", apperror.NewBadRequest(
			"wallet status must be active, frozen, or closed",
		)
	}
}

func normalizeEventLimit(limit int32) (int32, error) {
	if limit == 0 {
		return defaultEventLimit, nil
	}
	if limit < 1 || limit > maxEventLimit {
		return 0, apperror.NewBadRequest(
			"event limit must be between 1 and 200",
		)
	}

	return limit, nil
}

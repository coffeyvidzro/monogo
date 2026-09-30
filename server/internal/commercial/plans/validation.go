package plans

import (
	"fmt"
	"regexp"
	"strings"
)

var codePattern = regexp.MustCompile(`^[a-z][a-z0-9_]{0,63}$`)

func normalizeCreateRequest(req *CreateRequest) error {
	req.Code = strings.ToLower(strings.TrimSpace(req.Code))
	if !codePattern.MatchString(req.Code) {
		return fmt.Errorf("%w: plan code is invalid", ErrInvalidInput)
	}

	req.Name = strings.TrimSpace(req.Name)
	if len(req.Name) < 1 || len(req.Name) > 120 {
		return fmt.Errorf("%w: plan name must be between 1 and 120 characters", ErrInvalidInput)
	}

	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if req.Currency != "USD" {
		return fmt.Errorf("%w: currency must be USD", ErrInvalidInput)
	}

	if req.AmountMicros < 0 {
		return fmt.Errorf("%w: plan amount cannot be negative", ErrInvalidInput)
	}

	return nil
}

func normalizeCode(value string) string {
	return strings.ToLower(strings.TrimSpace(value))
}

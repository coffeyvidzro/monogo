package carriers

import (
	"net/netip"
	"strings"

	"github.com/google/uuid"
	"github.com/leamout/leamout/pkg/apperror"
)

func validateID(id uuid.UUID, field string) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest(field + " is required")
	}
	return nil
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 255 {
		return "", apperror.NewBadRequest("name must be between 1 and 255 characters")
	}
	return value, nil
}

func normalizeStatus(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value != "active" && value != "disabled" {
		return "", apperror.NewBadRequest("invalid status")
	}
	return value, nil
}

func validateLimits(cps, concurrent *int32, daily *int64) error {
	if cps != nil && *cps <= 0 {
		return apperror.NewBadRequest("max_cps must be greater than zero")
	}
	if concurrent != nil && *concurrent <= 0 {
		return apperror.NewBadRequest("max_concurrent_calls must be greater than zero")
	}
	if daily != nil && *daily <= 0 {
		return apperror.NewBadRequest("max_daily_minutes must be greater than zero")
	}
	return nil
}

var supportedCodecs = map[string]struct{}{"PCMU": {}, "PCMA": {}, "G722": {}, "G729": {}, "OPUS": {}}

func normalizeCodecs(values []string) ([]string, error) {
	if values == nil {
		return nil, nil
	}
	if len(values) == 0 {
		return nil, apperror.NewBadRequest("codecs must not be empty")
	}
	result, seen := make([]string, 0, len(values)), make(map[string]struct{}, len(values))
	for _, value := range values {
		value = strings.ToUpper(strings.TrimSpace(value))
		if _, ok := supportedCodecs[value]; !ok {
			return nil, apperror.NewBadRequest("unsupported codec: " + value)
		}
		if _, ok := seen[value]; !ok {
			result = append(result, value)
			seen[value] = struct{}{}
		}
	}
	return result, nil
}

func normalizeCIDR(value string) (netip.Prefix, error) {
	prefix, err := netip.ParsePrefix(strings.TrimSpace(value))
	if err != nil {
		return netip.Prefix{}, apperror.NewBadRequest("cidr must be a valid IPv4 or IPv6 network")
	}
	return prefix.Masked(), nil
}

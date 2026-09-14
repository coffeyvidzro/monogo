package trunks

import (
	"net"
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

func normalizeProvisioningMode(value ProvisioningMode) (ProvisioningMode, error) {
	value = ProvisioningMode(strings.ToLower(strings.TrimSpace(string(value))))
	if value == "" {
		return ProvisioningModeBYOC, nil
	}
	if value != ProvisioningModeBYOC && value != ProvisioningModeManaged {
		return "", apperror.NewBadRequest("type must be byoc or managed")
	}
	return value, nil
}

func normalizeName(value string) (string, error) {
	value = strings.TrimSpace(value)
	if value == "" || len(value) > 255 {
		return "", apperror.NewBadRequest("name must be between 1 and 255 characters")
	}
	return value, nil
}

func normalizeChoice(value string, choices map[string]struct{}, field string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if _, ok := choices[value]; !ok {
		return "", apperror.NewBadRequest("invalid " + field)
	}
	return value, nil
}

var directions = map[string]struct{}{"inbound": {}, "outbound": {}, "bidirectional": {}}
var statuses = map[string]struct{}{"active": {}, "disabled": {}}
var transports = map[string]struct{}{"udp": {}, "tcp": {}, "tls": {}}

func normalizeHost(value string) (string, error) {
	value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if value == "" || len(value) > 253 || strings.ContainsAny(value, " \t\r\n") {
		return "", apperror.NewBadRequest("host must be a valid IP address or hostname")
	}
	if net.ParseIP(value) != nil {
		return value, nil
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", apperror.NewBadRequest("host must be a valid IP address or hostname")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", apperror.NewBadRequest("host must be a valid IP address or hostname")
			}
		}
	}
	return value, nil
}

func validatePort(port int32) error {
	if port < 1 || port > 65535 {
		return apperror.NewBadRequest("port must be between 1 and 65535")
	}
	return nil
}

func validatePriority(priority int32) error {
	if priority < 0 {
		return apperror.NewBadRequest("priority must be non-negative")
	}
	return nil
}

func validateWeight(weight int32) error {
	if weight < 1 {
		return apperror.NewBadRequest("weight must be greater than zero")
	}
	return nil
}

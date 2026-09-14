package sip_domains

import (
	"net"
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
func validateDomainID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("sip domain id is required")
	}
	return nil
}
func normalizeDomain(value string) (string, error) {
	value = strings.ToLower(strings.TrimSuffix(strings.TrimSpace(value), "."))
	if value == "" || len(value) > 253 || net.ParseIP(value) != nil || !strings.Contains(value, ".") {
		return "", apperror.NewBadRequest("domain must be a valid fully qualified domain name")
	}
	for _, label := range strings.Split(value, ".") {
		if len(label) == 0 || len(label) > 63 || label[0] == '-' || label[len(label)-1] == '-' {
			return "", apperror.NewBadRequest("domain must be a valid fully qualified domain name")
		}
		for _, char := range label {
			if (char < 'a' || char > 'z') && (char < '0' || char > '9') && char != '-' {
				return "", apperror.NewBadRequest("domain must be a valid fully qualified domain name")
			}
		}
	}
	return value, nil
}

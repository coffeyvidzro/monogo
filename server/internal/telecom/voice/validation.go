package voice

import (
	"net/url"
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

func validateApplicationID(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("voice application id is required")
	}
	return nil
}

func normalizeName(value string) (string, error) {
	name := strings.TrimSpace(value)
	if name == "" {
		return "", apperror.NewBadRequest("name is required")
	}
	return name, nil
}

func normalizeOptionalString(value *string, field string) (*string, error) {
	if value == nil {
		return nil, nil
	}

	normalized := strings.TrimSpace(*value)
	if normalized == "" {
		return nil, apperror.NewBadRequest(field + " cannot be empty")
	}
	return &normalized, nil
}

func normalizeOptionalURL(value *string, field string) (*string, error) {
	normalized, err := normalizeOptionalString(value, field)
	if err != nil || normalized == nil {
		return normalized, err
	}

	parsed, err := url.ParseRequestURI(*normalized)
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return nil, apperror.NewBadRequest(field + " must be an http or https URL")
	}
	return normalized, nil
}

func validateRingTimeout(value *int32) error {
	if value == nil {
		return nil
	}
	if *value < 1 || *value > 300 {
		return apperror.NewBadRequest("ring_timeout_seconds must be between 1 and 300")
	}
	return nil
}

func validateBindingTarget(req CreateBindingRequest) error {
	targets := 0
	for _, id := range []*uuid.UUID{req.PhoneNumberID, req.SIPDomainID, req.SubscriberID} {
		if id != nil {
			if *id == uuid.Nil {
				return apperror.NewBadRequest("binding target id is required")
			}
			targets++
		}
	}
	if targets != 1 {
		return apperror.NewBadRequest("exactly one binding target is required")
	}
	return nil
}

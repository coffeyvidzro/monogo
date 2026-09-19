package numbers

import (
	"regexp"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

var e164 = regexp.MustCompile(`^\+[1-9][0-9]{6,14}$`)

func validateOrganization(id uuid.UUID) error {
	if id == uuid.Nil {
		return apperror.NewBadRequest("organization context required")
	}
	return nil
}

func validateIDs(organizationID, id uuid.UUID) error {
	if err := validateOrganization(organizationID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("number id is required")
	}
	return nil
}

func normalizeBYOC(req *CreateBYOCRequest) error {
	req.Number = strings.TrimSpace(req.Number)
	req.CountryCode = strings.ToUpper(strings.TrimSpace(req.CountryCode))
	if !e164.MatchString(req.Number) {
		return apperror.NewBadRequest("number must be in E.164 format")
	}
	if len(req.CountryCode) != 2 ||
		req.CountryCode[0] < 'A' || req.CountryCode[0] > 'Z' ||
		req.CountryCode[1] < 'A' || req.CountryCode[1] > 'Z' {
		return apperror.NewBadRequest("country_code must be a two-letter ISO country code")
	}
	if req.CarrierConnectionID != nil && *req.CarrierConnectionID == uuid.Nil {
		return apperror.NewBadRequest("carrier_connection_id must be a valid UUID")
	}
	return nil
}

func validateUpdate(req UpdateRequest) error {
	if req.VoiceEnabled == nil && req.SmsEnabled == nil {
		return apperror.NewBadRequest("at least one capability is required")
	}
	return nil
}

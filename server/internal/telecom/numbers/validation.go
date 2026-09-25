package numbers

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
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

func normalizeManagedPurchase(organizationID uuid.UUID, key string, req *ManagedPurchaseRequest) error {
	if err := validateOrganization(organizationID); err != nil {
		return err
	}
	if key == "" || len(key) > 255 {
		return apperror.NewBadRequest("Idempotency-Key is required and must not exceed 255 characters")
	}
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
	return nil
}

func normalizeProviderNumber(number string) string {
	number = strings.TrimSpace(number)
	if !strings.HasPrefix(number, "+") {
		return "+" + number
	}
	return number
}

func availableDIDSKU(available didww.AvailableDID) (string, error) {
	for _, key := range []string{"sku", "did_group"} {
		raw, ok := available.Relationships[key]
		if !ok {
			continue
		}
		var relationship struct {
			Data *didww.ResourceIdentifier `json:"data"`
		}
		if err := json.Unmarshal(raw, &relationship); err == nil &&
			relationship.Data != nil && strings.TrimSpace(relationship.Data.ID) != "" {
			return relationship.Data.ID, nil
		}
	}
	return "", fmt.Errorf("available DID %q has no SKU relationship", available.ID)
}

func didUsesTrunk(did didww.DID, trunkID string) bool {
	relationship, ok := did.Relationships["voice_in_trunk"]
	return ok && relationship.Data != nil &&
		relationship.Data.Type == "voice_in_trunks" && relationship.Data.ID == trunkID
}

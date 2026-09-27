package charges

import (
	"encoding/json"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

func normalizeCreate(req *CreateRequest) error {
	resourceType, err := normalizeResourceType(
		req.ResourceType,
	)
	if err != nil {
		return err
	}

	req.ResourceType = resourceType
	req.ChargingMode = strings.ToLower(
		strings.TrimSpace(req.ChargingMode),
	)
	req.Currency = strings.ToUpper(
		strings.TrimSpace(req.Currency),
	)
	req.IdempotencyKey = strings.TrimSpace(
		req.IdempotencyKey,
	)
	req.RequestHash = strings.ToLower(
		strings.TrimSpace(req.RequestHash),
	)

	if req.OrganizationID == uuid.Nil ||
		req.WalletID == uuid.Nil ||
		req.ResourceID == uuid.Nil {
		return apperror.NewBadRequest(
			"organization, wallet, and resource are required",
		)
	}

	if req.ChargingMode != ModeRolling &&
		req.ChargingMode != ModeDiscrete {
		return apperror.NewBadRequest(
			"charging_mode must be rolling or discrete",
		)
	}

	if err := validateCurrency(req.Currency); err != nil {
		return err
	}

	if req.IdempotencyKey == "" ||
		len(req.IdempotencyKey) > 255 {
		return apperror.NewBadRequest(
			"idempotency key must be between 1 and 255 characters",
		)
	}

	if err := validateRequestHash(req.RequestHash); err != nil {
		return err
	}

	if len(req.PricingSnapshot) == 0 {
		req.PricingSnapshot = json.RawMessage("{}")
	}

	var snapshot map[string]any
	if err := json.Unmarshal(
		req.PricingSnapshot,
		&snapshot,
	); err != nil || snapshot == nil {
		return apperror.NewBadRequest(
			"pricing_snapshot must be a JSON object",
		)
	}

	return nil
}

func normalizeResourceType(value string) (string, error) {
	value = strings.ToLower(
		strings.TrimSpace(value),
	)

	if value == "" || len(value) > 64 {
		return "", apperror.NewBadRequest(
			"resource_type must be between 1 and 64 characters",
		)
	}

	for index, character := range value {
		validCharacter := (character >= 'a' && character <= 'z') ||
			(character >= '0' && character <= '9') ||
			character == '_'
		if !validCharacter {
			return "", apperror.NewBadRequest(
				"resource_type must use lowercase letters, digits, or underscores",
			)
		}

		if index == 0 &&
			(character < 'a' || character > 'z') {
			return "", apperror.NewBadRequest(
				"resource_type must start with a lowercase letter",
			)
		}
	}

	return value, nil
}

func validateCurrency(value string) error {
	if len(value) != 3 {
		return apperror.NewBadRequest(
			"currency must be a 3-letter ISO code",
		)
	}

	for _, character := range value {
		if character < 'A' || character > 'Z' {
			return apperror.NewBadRequest(
				"currency must be a 3-letter ISO code",
			)
		}
	}

	return nil
}

func validateRequestHash(value string) error {
	if len(value) != 64 {
		return apperror.NewBadRequest(
			"request hash must be a 64-character lowercase hex digest",
		)
	}

	for _, character := range value {
		isDigit := character >= '0' && character <= '9'
		isHexLetter := character >= 'a' && character <= 'f'
		if !isDigit && !isHexLetter {
			return apperror.NewBadRequest(
				"request hash must be a 64-character lowercase hex digest",
			)
		}
	}

	return nil
}

func normalizeListRequest(req *ListRequest) error {
	if req.Offset < 0 {
		return apperror.NewBadRequest(
			"offset cannot be negative",
		)
	}

	if req.Limit == 0 {
		req.Limit = defaultListLimit
	}
	if req.Limit < 1 || req.Limit > maxListLimit {
		return apperror.NewBadRequest(
			"limit must be between 1 and 200",
		)
	}

	if req.Status == nil {
		return nil
	}

	status := strings.ToLower(
		strings.TrimSpace(*req.Status),
	)

	switch status {
	case "pending":
	case "active":
	case "completed":
	case "failed":
	case "cancelled":
	default:
		return apperror.NewBadRequest(
			"invalid charge status",
		)
	}

	req.Status = &status

	return nil
}

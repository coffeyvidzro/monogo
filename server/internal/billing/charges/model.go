package charges

import (
	"encoding/json"
	"strings"

	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

const (
	ModeRolling  = "rolling"
	ModeDiscrete = "discrete"
)

type CreateRequest struct {
	OrganizationID  uuid.UUID
	WalletID        uuid.UUID
	ResourceType    string
	ResourceID      uuid.UUID
	ChargingMode    string
	Currency        string
	IdempotencyKey  string
	RequestHash     string
	PricingSnapshot json.RawMessage
}

type ListRequest struct {
	Status *string
	Offset int32
	Limit  int32
}

func normalizeCreate(req *CreateRequest) error {
	resourceType, err := normalizeResourceType(req.ResourceType)
	if err != nil {
		return err
	}
	req.ResourceType = resourceType
	req.ChargingMode = strings.ToLower(strings.TrimSpace(req.ChargingMode))
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	req.IdempotencyKey = strings.TrimSpace(req.IdempotencyKey)
	req.RequestHash = strings.ToLower(strings.TrimSpace(req.RequestHash))

	if req.OrganizationID == uuid.Nil || req.WalletID == uuid.Nil || req.ResourceID == uuid.Nil {
		return apperror.NewBadRequest("organization, wallet, and resource are required")
	}
	if req.ChargingMode != ModeRolling && req.ChargingMode != ModeDiscrete {
		return apperror.NewBadRequest("charging_mode must be rolling or discrete")
	}
	if len(req.Currency) != 3 {
		return apperror.NewBadRequest("currency must be a 3-letter ISO code")
	}
	for _, r := range req.Currency {
		if r < 'A' || r > 'Z' {
			return apperror.NewBadRequest("currency must be a 3-letter ISO code")
		}
	}
	if req.IdempotencyKey == "" || len(req.IdempotencyKey) > 255 {
		return apperror.NewBadRequest("idempotency key must be between 1 and 255 characters")
	}
	if len(req.RequestHash) != 64 {
		return apperror.NewBadRequest("request hash must be a 64-character lowercase hex digest")
	}
	for _, r := range req.RequestHash {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return apperror.NewBadRequest("request hash must be a 64-character lowercase hex digest")
		}
	}
	if len(req.PricingSnapshot) == 0 {
		req.PricingSnapshot = json.RawMessage("{}")
	}
	var value map[string]any
	if err := json.Unmarshal(req.PricingSnapshot, &value); err != nil || value == nil {
		return apperror.NewBadRequest("pricing_snapshot must be a JSON object")
	}
	return nil
}

func normalizeResourceType(value string) (string, error) {
	value = strings.ToLower(strings.TrimSpace(value))
	if value == "" || len(value) > 64 {
		return "", apperror.NewBadRequest("resource_type must be between 1 and 64 characters")
	}
	for i, r := range value {
		if (r < 'a' || r > 'z') && (r < '0' || r > '9') && r != '_' {
			return "", apperror.NewBadRequest(
				"resource_type must use lowercase letters, digits, or underscores",
			)
		}
		if i == 0 && (r < 'a' || r > 'z') {
			return "", apperror.NewBadRequest("resource_type must start with a lowercase letter")
		}
	}
	return value, nil
}

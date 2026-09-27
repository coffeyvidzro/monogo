package charges

import (
	"encoding/json"

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

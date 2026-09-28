package authorization

import (
	"time"

	"github.com/google/uuid"
)

type CallAuthorization struct {
	CallID                  uuid.UUID
	OrganizationID          uuid.UUID
	CarrierRateID           uuid.UUID
	Currency                string
	RateMicros              int64
	BillingUnit             string
	BillingIncrementSeconds int32
	MinimumDurationSeconds  int32
	AuthorizedAt            time.Time
}

type ManagedCallRequest struct {
	CallID         uuid.UUID
	OrganizationID uuid.UUID
	Destination    string
	Direction      string
	RequestedAt    time.Time
}

package authorization

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	ProvisioningBYOC    = "byoc"
	ProvisioningManaged = "managed"

	CurrencyUSD = "USD"
)

var ErrInvalidInput = errors.New("invalid commercial authorization input")

type CallRequest struct {
	OrganizationID   uuid.UUID
	OperationID      uuid.UUID
	Destination      string
	Direction        string
	ProvisioningMode string
	MinimumSeconds   int64
	RequestedAt      time.Time
}

type Decision struct {
	OperationID            uuid.UUID  `json:"operation_id"`
	Billable               bool       `json:"billable"`
	Currency               *string    `json:"currency,omitempty"`
	RateID                 *uuid.UUID `json:"rate_id,omitempty"`
	RateMicros             int64      `json:"rate_micros"`
	AuthorizedAmountMicros int64      `json:"authorized_amount_micros"`
}

type CaptureRequest struct {
	OrganizationID  uuid.UUID
	OperationID     uuid.UUID
	ReferenceID     uuid.UUID
	BillableSeconds int64
	Decision        Decision
	OccurredAt      time.Time
}

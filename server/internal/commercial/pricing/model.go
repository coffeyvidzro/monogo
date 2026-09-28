package pricing

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
	BillingUnitMinute = "minute"
)

var (
	ErrInvalidInput = errors.New("invalid pricing input")
	ErrRateNotFound = errors.New("carrier rate not found")
	ErrRateConflict = errors.New("carrier rate conflict")
)

type Rate struct {
	ID                      uuid.UUID  `json:"id"`
	OrganizationID          *uuid.UUID `json:"organization_id,omitempty"`
	DestinationPrefix       string     `json:"destination_prefix"`
	Direction               string     `json:"direction"`
	Currency                string     `json:"currency"`
	RateMicros              int64      `json:"rate_micros"`
	BillingUnit             string     `json:"billing_unit"`
	BillingIncrementSeconds int32      `json:"billing_increment_seconds"`
	MinimumDurationSeconds  int32      `json:"minimum_duration_seconds"`
	EffectiveAt             time.Time  `json:"effective_at"`
	ExpiresAt               *time.Time `json:"expires_at,omitempty"`
	CreatedAt               time.Time  `json:"created_at"`
}

type CreateRateRequest struct {
	OrganizationID          *uuid.UUID
	DestinationPrefix       string
	Direction               string
	Currency                string
	RateMicros              int64
	BillingUnit             string
	BillingIncrementSeconds int32
	MinimumDurationSeconds  int32
	EffectiveAt             time.Time
	ExpiresAt               *time.Time
}

type ResolveRequest struct {
	OrganizationID    uuid.UUID
	DestinationDigits string
	Direction         string
	Currency          string
	ResolvedAt        time.Time
}

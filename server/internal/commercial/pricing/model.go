package pricing

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

var (
	ErrInvalidInput = errors.New("invalid pricing input")
	ErrRateNotFound = errors.New("carrier rate not found")
	ErrRateConflict = errors.New("carrier rate conflict")
)

type Rate struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    *uuid.UUID `json:"organization_id,omitempty"`
	DestinationPrefix string     `json:"destination_prefix"`
	Direction         string     `json:"direction"`
	Currency          string     `json:"currency"`
	RateMicros        int64      `json:"rate_micros"`
	EffectiveAt       time.Time  `json:"effective_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type CreateRateRequest struct {
	OrganizationID    *uuid.UUID
	DestinationPrefix string
	Direction         string
	Currency          string
	RateMicros        int64
	EffectiveAt       time.Time
	ExpiresAt         *time.Time
}

type ResolveRequest struct {
	OrganizationID    uuid.UUID
	DestinationDigits string
	Direction         string
	Currency          string
	ResolvedAt        time.Time
}

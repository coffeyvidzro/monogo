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
	ErrRateNotFound = errors.New("carrier rate not found")
	ErrRateConflict = errors.New("carrier rate conflict")
)

type Rate struct {
	ID                uuid.UUID
	OrganizationID    *uuid.UUID
	DestinationPrefix string
	Direction         string
	Currency          string
	RateMicros        int64
	EffectiveAt       time.Time
	ExpiresAt         *time.Time
	CreatedAt         time.Time
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

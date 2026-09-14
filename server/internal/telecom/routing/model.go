package routing

import (
	"errors"

	"github.com/google/uuid"
)

var (
	ErrNoRoute         = errors.New("no route available")
	ErrCarrierMismatch = errors.New("carrier connection does not own phone number")
	ErrTenantMismatch  = errors.New("route organization mismatch")
	ErrCallerIdentity  = errors.New("caller identity is not authorized for carrier route")
)

type OutboundRequest struct {
	OrganizationID uuid.UUID
	ApplicationID  *uuid.UUID
	From           string
	To             string
	TrunkID        *uuid.UUID
}

type OutboundDecision struct {
	Managed             bool
	OrganizationID      uuid.UUID
	CarrierConnectionID uuid.UUID
	TrunkID             uuid.UUID
	EndpointID          uuid.UUID
	Host                string
	Port                int32
	Transport           string
	From                string
	To                  string
	MaxCPS              int32
	MaxConcurrentCalls  int32
	MaxDailyMinutes     *int64
}

type InboundRequest struct {
	SourceIP     string
	CalledNumber string
	CallerNumber string
}

type InboundDecision struct {
	OrganizationID      uuid.UUID
	CarrierConnectionID uuid.UUID
	PhoneNumberID       uuid.UUID
	VoiceApplicationID  uuid.UUID
	CalledNumber        string
	CallerNumber        string
}

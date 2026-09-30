package routing

import (
	"github.com/google/uuid"
)

type InboundRequest struct {
	OrganizationID      uuid.UUID
	ApplicationID       uuid.UUID
	PhoneNumberID       uuid.UUID
	VoiceBindingID      uuid.UUID
	CarrierConnectionID uuid.UUID
	CalledNumber        string
}

type Limits struct {
	MaxCPS             int32
	MaxConcurrentCalls int32
}

type InboundDecision struct {
	OrganizationID      uuid.UUID
	ApplicationID       uuid.UUID
	PhoneNumberID       uuid.UUID
	VoiceBindingID      uuid.UUID
	CarrierConnectionID uuid.UUID
	CalledNumber        string
	Limits              Limits
}

type OutboundRequest struct {
	OrganizationID uuid.UUID
	TrunkID        *uuid.UUID
	Destination    string
}

type OutboundRoute struct {
	CarrierConnectionID uuid.UUID
	TrunkID             uuid.UUID
	TrunkEndpointID     uuid.UUID
	Host                string
	Port                uint16
	Transport           string
	Limits              Limits
}

type OutboundDecision struct {
	DestinationDigits string
	Routes            []OutboundRoute
}

func (d OutboundDecision) Primary() (OutboundRoute, bool) {
	if len(d.Routes) == 0 {
		return OutboundRoute{}, false
	}
	return d.Routes[0], true
}

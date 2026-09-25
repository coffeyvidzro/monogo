package routing

import (
	"time"

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
	MaxDailyMinutes    *int64
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
	ProvisioningMode    string
	Host                string
	Port                uint16
	Transport           string
	Limits              Limits
	RateMicros          int64
	ScoreMicros         int64
}

type OutboundDecision struct {
	ID     uuid.UUID
	Routes []OutboundRoute
}

func (d OutboundDecision) Primary() (OutboundRoute, bool) {
	if len(d.Routes) == 0 {
		return OutboundRoute{}, false
	}
	return d.Routes[0], true
}

type managedRouteCandidate struct {
	Candidate             CarrierCandidate
	Route                 OutboundRoute
	ASRBasisPoints        int32
	ALOCMilliseconds      int64
	LatencyMilliseconds   int32
	PacketLossBasisPoints int32
	MetricsObservedAt     time.Time
}

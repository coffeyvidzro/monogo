package routing

import "github.com/google/uuid"

type InboundRequest struct {
	OrganizationID      uuid.UUID
	ApplicationID       uuid.UUID
	PhoneNumberID       uuid.UUID
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
	CarrierConnectionID uuid.UUID
	CalledNumber        string
	Limits              Limits
}

type OutboundRequest struct {
	OrganizationID uuid.UUID
	TrunkID        *uuid.UUID
	Destination    string
}

type OutboundDecision struct {
	CarrierConnectionID uuid.UUID
	TrunkID             uuid.UUID
	TrunkEndpointID     uuid.UUID
	ProvisioningMode    string
	Host                string
	Port                uint16
	Transport           string
}

package trunks

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type ProvisioningMode string

const (
	ProvisioningModeBYOC    ProvisioningMode = "byoc"
	ProvisioningModeManaged ProvisioningMode = "managed"
	ManagedVoiceEntitlement                  = "voice.managed.enabled"
	ManagedSIPHost                           = "sip.leamout.com"
	ManagedSIPPort          int32            = 5061
	ManagedSIPTransport                      = "tls"
	ManagedSIPRealm                          = "sip.leamout.com"
)

type CreateRequest struct {
	Type                ProvisioningMode `json:"type"`
	CarrierConnectionID *uuid.UUID       `json:"carrier_connection_id,omitempty"`
	Name                string           `json:"name"`
	Direction           *string          `json:"direction,omitempty"`
	Status              *string          `json:"status,omitempty"`
}

type UpdateRequest struct {
	Name      *string `json:"name,omitempty"`
	Direction *string `json:"direction,omitempty"`
	Status    *string `json:"status,omitempty"`
}

type EndpointCreateRequest struct {
	Host      string  `json:"host"`
	Port      *int32  `json:"port,omitempty"`
	Transport *string `json:"transport,omitempty"`
	Direction *string `json:"direction,omitempty"`
	Priority  *int32  `json:"priority,omitempty"`
	Weight    *int32  `json:"weight,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

type EndpointUpdateRequest struct {
	Host      *string `json:"host,omitempty"`
	Port      *int32  `json:"port,omitempty"`
	Transport *string `json:"transport,omitempty"`
	Direction *string `json:"direction,omitempty"`
	Priority  *int32  `json:"priority,omitempty"`
	Weight    *int32  `json:"weight,omitempty"`
	Enabled   *bool   `json:"enabled,omitempty"`
}

type SIPCredential struct {
	Host      string `json:"host"`
	Port      int32  `json:"port"`
	Transport string `json:"transport"`
	Realm     string `json:"realm"`
	Username  string `json:"username"`
	Password  string `json:"password"`
}

type CreateResult struct {
	Trunk      sqlc.Trunk
	Credential *SIPCredential
}

type EventType string

const (
	EventTrunkCreated           EventType = "trunk.created"
	EventTrunkUpdated           EventType = "trunk.updated"
	EventTrunkDisabled          EventType = "trunk.disabled"
	EventTrunkCredentialRotated EventType = "trunk.credential.rotated"
	EventTrunkEndpointCreated   EventType = "trunk.endpoint.created"
	EventTrunkEndpointUpdated   EventType = "trunk.endpoint.updated"
	EventTrunkEndpointDeleted   EventType = "trunk.endpoint.deleted"
)

type Event struct {
	EventType      EventType  `json:"event_type"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	TrunkID        uuid.UUID  `json:"trunk_id"`
	EndpointID     *uuid.UUID `json:"endpoint_id,omitempty"`
	Resource       any        `json:"resource"`
	OccurredAt     time.Time  `json:"occurred_at"`
}

type Response struct {
	ID                  uuid.UUID        `json:"id"`
	OrganizationID      *uuid.UUID       `json:"organization_id,omitempty"`
	Type                ProvisioningMode `json:"type"`
	CarrierConnectionID *uuid.UUID       `json:"carrier_connection_id,omitempty"`
	Name                string           `json:"name"`
	Direction           string           `json:"direction"`
	Status              string           `json:"status"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

type ManagedCreateResponse struct {
	Response
	SIP SIPCredential `json:"sip"`
}

type EndpointResponse struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      *uuid.UUID `json:"organization_id,omitempty"`
	TrunkID             uuid.UUID  `json:"trunk_id"`
	Host                string     `json:"host"`
	Port                int32      `json:"port"`
	Transport           string     `json:"transport"`
	Direction           string     `json:"direction"`
	Priority            int32      `json:"priority"`
	Weight              int32      `json:"weight"`
	Enabled             bool       `json:"enabled"`
	HealthStatus        string     `json:"health_status"`
	ConsecutiveFailures int32      `json:"consecutive_failures"`
	LastCheckedAt       *time.Time `json:"last_checked_at,omitempty"`
	LastResponseCode    *int32     `json:"last_response_code,omitempty"`
	LastLatencyMs       *int32     `json:"last_latency_ms,omitempty"`
	LastError           *string    `json:"last_error,omitempty"`
	CooldownUntil       *time.Time `json:"cooldown_until,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func response(trunk sqlc.Trunk) Response {
	carrierConnectionID := trunk.CarrierConnectionID
	if ProvisioningMode(trunk.ProvisioningMode) == ProvisioningModeManaged {
		carrierConnectionID = nil
	}
	return Response{
		ID:                  trunk.ID,
		OrganizationID:      trunk.OrganizationID,
		Type:                ProvisioningMode(trunk.ProvisioningMode),
		CarrierConnectionID: carrierConnectionID,
		Name:                trunk.Name,
		Direction:           trunk.Direction,
		Status:              trunk.Status,
		CreatedAt:           pgconv.TimestamptzToTime(trunk.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(trunk.UpdatedAt),
	}
}

func endpointResponse(endpoint sqlc.TrunkEndpoint) EndpointResponse {
	return EndpointResponse{
		ID: endpoint.ID, OrganizationID: endpoint.OrganizationID, TrunkID: endpoint.TrunkID,
		Host: endpoint.Host, Port: endpoint.Port, Transport: endpoint.Transport, Direction: endpoint.Direction,
		Priority: endpoint.Priority, Weight: endpoint.Weight, Enabled: endpoint.Enabled,
		HealthStatus: endpoint.HealthStatus, ConsecutiveFailures: endpoint.ConsecutiveFailures,
		LastCheckedAt: pgconv.TimestamptzToTimePtr(endpoint.LastCheckedAt), LastResponseCode: endpoint.LastResponseCode,
		LastLatencyMs: endpoint.LastLatencyMs, LastError: endpoint.LastError,
		CooldownUntil: pgconv.TimestamptzToTimePtr(endpoint.CooldownUntil),
		CreatedAt:     pgconv.TimestamptzToTime(endpoint.CreatedAt), UpdatedAt: pgconv.TimestamptzToTime(endpoint.UpdatedAt),
	}
}

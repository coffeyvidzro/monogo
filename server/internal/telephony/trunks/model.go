package trunks

import (
	"net/netip"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type DigestCredential struct {
	Username string `json:"username"`
	Realm    string `json:"realm"`
	Secret   string `json:"secret"`
}

type CreateRequest struct {
	Name               string            `json:"name"`
	Direction          *string           `json:"direction,omitempty"`
	Status             *string           `json:"status,omitempty"`
	OutboundCredential *DigestCredential `json:"outbound_credential,omitempty"`
	InboundEnabled     *bool             `json:"inbound_enabled,omitempty"`
	InboundAuthMethod  *string           `json:"inbound_auth_method,omitempty"`
	InboundCredential  *DigestCredential `json:"inbound_credential,omitempty"`
	MaxCPS             *int32            `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32            `json:"max_concurrent_calls,omitempty"`
	Codecs             []string          `json:"codecs,omitempty"`
	SupportsVideo      *bool             `json:"supports_video,omitempty"`
	SupportsFax        *bool             `json:"supports_fax,omitempty"`
}

type UpdateRequest struct {
	Name               *string   `json:"name,omitempty"`
	Direction          *string   `json:"direction,omitempty"`
	Status             *string   `json:"status,omitempty"`
	InboundEnabled     *bool     `json:"inbound_enabled,omitempty"`
	MaxCPS             *int32    `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32    `json:"max_concurrent_calls,omitempty"`
	Codecs             *[]string `json:"codecs,omitempty"`
	SupportsVideo      *bool     `json:"supports_video,omitempty"`
	SupportsFax        *bool     `json:"supports_fax,omitempty"`
}

type AuthRequest struct {
	Method   string  `json:"method"`
	Username *string `json:"username,omitempty"`
	Realm    *string `json:"realm,omitempty"`
	Secret   *string `json:"secret,omitempty"`
}

type SourceIPRequest struct {
	CIDR string `json:"cidr"`
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

type EventType string

const (
	EventTrunkCreated         EventType = "trunk.created"
	EventTrunkUpdated         EventType = "trunk.updated"
	EventTrunkDisabled        EventType = "trunk.disabled"
	EventTrunkEndpointCreated EventType = "trunk.endpoint.created"
	EventTrunkEndpointUpdated EventType = "trunk.endpoint.updated"
	EventTrunkEndpointDeleted EventType = "trunk.endpoint.deleted"
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
	ID                     uuid.UUID `json:"id"`
	OrganizationID         uuid.UUID `json:"organization_id"`
	Name                   string    `json:"name"`
	Direction              string    `json:"direction"`
	Status                 string    `json:"status"`
	OutboundAuthMethod     string    `json:"outbound_auth_method"`
	OutboundUsername       *string   `json:"outbound_username,omitempty"`
	OutboundRealm          *string   `json:"outbound_realm,omitempty"`
	HasOutboundCredentials bool      `json:"has_outbound_credentials"`
	InboundEnabled         bool      `json:"inbound_enabled"`
	InboundAuthMethod      string    `json:"inbound_auth_method"`
	InboundUsername        *string   `json:"inbound_username,omitempty"`
	InboundRealm           *string   `json:"inbound_realm,omitempty"`
	HasInboundCredentials  bool      `json:"has_inbound_credentials"`
	MaxCPS                 int32     `json:"max_cps"`
	MaxConcurrentCalls     int32     `json:"max_concurrent_calls"`
	Codecs                 []string  `json:"codecs"`
	SupportsVideo          bool      `json:"supports_video"`
	SupportsFax            bool      `json:"supports_fax"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SourceIPResponse struct {
	ID        uuid.UUID    `json:"id"`
	TrunkID   uuid.UUID    `json:"trunk_id"`
	CIDR      netip.Prefix `json:"cidr"`
	CreatedAt time.Time    `json:"created_at"`
}

type EndpointResponse struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
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

type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

func response(trunk sqlc.Trunk) Response {
	return Response{
		ID:                     trunk.ID,
		OrganizationID:         trunk.OrganizationID,
		Name:                   trunk.Name,
		Direction:              trunk.Direction,
		Status:                 trunk.Status,
		OutboundAuthMethod:     trunk.OutboundAuthMethod,
		OutboundUsername:       trunk.AuthUsername,
		OutboundRealm:          trunk.AuthRealm,
		HasOutboundCredentials: trunk.AuthSecretCiphertext != nil,
		InboundEnabled:         trunk.InboundEnabled,
		InboundAuthMethod:      trunk.InboundAuthMethod,
		InboundUsername:        trunk.InboundUsername,
		InboundRealm:           trunk.InboundRealm,
		HasInboundCredentials:  trunk.InboundSecretCiphertext != nil,
		MaxCPS:                 trunk.MaxCps,
		MaxConcurrentCalls:     trunk.MaxConcurrentCalls,
		Codecs:                 trunk.Codecs,
		SupportsVideo:          trunk.SupportsVideo,
		SupportsFax:            trunk.SupportsFax,
		CreatedAt:              pgconv.TimestamptzToTime(trunk.CreatedAt),
		UpdatedAt:              pgconv.TimestamptzToTime(trunk.UpdatedAt),
	}
}

func sourceIPResponse(row sqlc.TrunkSourceIp) SourceIPResponse {
	return SourceIPResponse{
		ID:        row.ID,
		TrunkID:   row.TrunkID,
		CIDR:      row.Cidr,
		CreatedAt: pgconv.TimestamptzToTime(row.CreatedAt),
	}
}

func endpointResponse(endpoint sqlc.TrunkEndpoint) EndpointResponse {
	return EndpointResponse{
		ID:                  endpoint.ID,
		OrganizationID:      endpoint.OrganizationID,
		TrunkID:             endpoint.TrunkID,
		Host:                endpoint.Host,
		Port:                endpoint.Port,
		Transport:           endpoint.Transport,
		Direction:           endpoint.Direction,
		Priority:            endpoint.Priority,
		Weight:              endpoint.Weight,
		Enabled:             endpoint.Enabled,
		HealthStatus:        endpoint.HealthStatus,
		ConsecutiveFailures: endpoint.ConsecutiveFailures,
		LastCheckedAt:       pgconv.TimestamptzToTimePtr(endpoint.LastCheckedAt),
		LastResponseCode:    endpoint.LastResponseCode,
		LastLatencyMs:       endpoint.LastLatencyMs,
		LastError:           endpoint.LastError,
		CooldownUntil:       pgconv.TimestamptzToTimePtr(endpoint.CooldownUntil),
		CreatedAt:           pgconv.TimestamptzToTime(endpoint.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(endpoint.UpdatedAt),
	}
}

package carriers

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type CreateRequest struct {
	ProviderID         uuid.UUID `json:"provider_id"`
	Name               string    `json:"name"`
	Status             *string   `json:"status,omitempty"`
	InboundEnabled     *bool     `json:"inbound_enabled,omitempty"`
	MaxCPS             *int32    `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32    `json:"max_concurrent_calls,omitempty"`
	MaxDailyMinutes    *int64    `json:"max_daily_minutes,omitempty"`
	Codecs             []string  `json:"codecs,omitempty"`
	SupportsVideo      *bool     `json:"supports_video,omitempty"`
	SupportsFax        *bool     `json:"supports_fax,omitempty"`
}

type UpdateRequest struct {
	Name               *string  `json:"name,omitempty"`
	Status             *string  `json:"status,omitempty"`
	InboundEnabled     *bool    `json:"inbound_enabled,omitempty"`
	MaxCPS             *int32   `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32   `json:"max_concurrent_calls,omitempty"`
	MaxDailyMinutes    *int64   `json:"max_daily_minutes,omitempty"`
	Codecs             []string `json:"codecs,omitempty"`
	SupportsVideo      *bool    `json:"supports_video,omitempty"`
	SupportsFax        *bool    `json:"supports_fax,omitempty"`
}

type SourceIPCreateRequest struct {
	CIDR string `json:"cidr"`
}

type DigestAuthRequest struct {
	Method   string `json:"method"`
	Username string `json:"username"`
	Realm    string `json:"realm"`
	Secret   string `json:"secret"`
}

type InboundAuthRequest struct {
	Method   string `json:"method"`
	Username string `json:"username,omitempty"`
	Realm    string `json:"realm,omitempty"`
	Secret   string `json:"secret,omitempty"`
}

type ValidationIssue struct {
	Severity   string     `json:"severity"`
	Code       string     `json:"code"`
	Message    string     `json:"message"`
	TrunkID    *uuid.UUID `json:"trunk_id,omitempty"`
	EndpointID *uuid.UUID `json:"endpoint_id,omitempty"`
}

type EndpointValidation struct {
	TrunkID      uuid.UUID `json:"trunk_id"`
	EndpointID   uuid.UUID `json:"endpoint_id"`
	Host         string    `json:"host"`
	Port         int32     `json:"port"`
	Transport    string    `json:"transport"`
	Reachable    bool      `json:"reachable"`
	ResponseCode *int32    `json:"response_code,omitempty"`
	LatencyMS    int64     `json:"latency_ms"`
	Error        string    `json:"error,omitempty"`
}

type ValidationResponse struct {
	CarrierConnectionID uuid.UUID            `json:"carrier_connection_id"`
	Valid               bool                 `json:"valid"`
	Issues              []ValidationIssue    `json:"issues"`
	Endpoints           []EndpointValidation `json:"endpoints"`
}

type Response struct {
	ID                     uuid.UUID  `json:"id"`
	OrganizationID         *uuid.UUID `json:"organization_id,omitempty"`
	ProviderID             uuid.UUID  `json:"provider_id"`
	Name                   string     `json:"name"`
	Status                 string     `json:"status"`
	OutboundAuthMethod     string     `json:"outbound_auth_method"`
	AuthUsername           *string    `json:"auth_username,omitempty"`
	HasOutboundCredentials bool       `json:"has_outbound_credentials"`
	InboundEnabled         bool       `json:"inbound_enabled"`
	InboundAuthMethod      string     `json:"inbound_auth_method"`
	InboundUsername        *string    `json:"inbound_username,omitempty"`
	HasInboundCredentials  bool       `json:"has_inbound_credentials"`
	MaxCPS                 int32      `json:"max_cps"`
	MaxConcurrentCalls     int32      `json:"max_concurrent_calls"`
	MaxDailyMinutes        *int64     `json:"max_daily_minutes,omitempty"`
	Codecs                 []string   `json:"codecs"`
	SupportsVideo          bool       `json:"supports_video"`
	SupportsFax            bool       `json:"supports_fax"`
	CreatedAt              time.Time  `json:"created_at"`
	UpdatedAt              time.Time  `json:"updated_at"`
}

type SourceIPResponse struct {
	ID                  uuid.UUID    `json:"id"`
	OrganizationID      *uuid.UUID   `json:"organization_id,omitempty"`
	CarrierConnectionID uuid.UUID    `json:"carrier_connection_id"`
	CIDR                netip.Prefix `json:"cidr"`
	CreatedAt           time.Time    `json:"created_at"`
}

type ProviderResponse struct {
	ID        uuid.UUID `json:"id"`
	Slug      string    `json:"slug"`
	Name      string    `json:"name"`
	Adapter   string    `json:"adapter"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

func response(connection sqlc.CarrierConnection) Response {
	return Response{
		ID:                     connection.ID,
		OrganizationID:         connection.OrganizationID,
		ProviderID:             connection.ProviderID,
		Name:                   connection.Name,
		Status:                 connection.Status,
		OutboundAuthMethod:     connection.OutboundAuthMethod,
		AuthUsername:           connection.AuthUsername,
		HasOutboundCredentials: connection.AuthSecretCiphertext != nil,
		InboundEnabled:         connection.InboundEnabled,
		InboundAuthMethod:      connection.InboundAuthMethod,
		InboundUsername:        connection.InboundUsername,
		HasInboundCredentials:  connection.InboundSecretCiphertext != nil,
		MaxCPS:                 connection.MaxCps,
		MaxConcurrentCalls:     connection.MaxConcurrentCalls,
		MaxDailyMinutes:        connection.MaxDailyMinutes,
		Codecs:                 connection.Codecs,
		SupportsVideo:          connection.SupportsVideo,
		SupportsFax:            connection.SupportsFax,
		CreatedAt:              pgconv.TimestamptzToTime(connection.CreatedAt),
		UpdatedAt:              pgconv.TimestamptzToTime(connection.UpdatedAt),
	}
}

func getResponse(connection sqlc.GetCarrierConnectionByIDRow) Response {
	return Response{
		ID:                     connection.ID,
		OrganizationID:         connection.OrganizationID,
		ProviderID:             connection.ProviderID,
		Name:                   connection.Name,
		Status:                 connection.Status,
		OutboundAuthMethod:     connection.OutboundAuthMethod,
		AuthUsername:           connection.AuthUsername,
		HasOutboundCredentials: boolValue(connection.HasOutboundCredentials),
		InboundEnabled:         connection.InboundEnabled,
		InboundAuthMethod:      connection.InboundAuthMethod,
		InboundUsername:        connection.InboundUsername,
		HasInboundCredentials:  boolValue(connection.HasInboundCredentials),
		MaxCPS:                 connection.MaxCps,
		MaxConcurrentCalls:     connection.MaxConcurrentCalls,
		MaxDailyMinutes:        connection.MaxDailyMinutes,
		Codecs:                 connection.Codecs,
		SupportsVideo:          connection.SupportsVideo,
		SupportsFax:            connection.SupportsFax,
		CreatedAt:              pgconv.TimestamptzToTime(connection.CreatedAt),
		UpdatedAt:              pgconv.TimestamptzToTime(connection.UpdatedAt),
	}
}

func listResponse(connection sqlc.ListCarrierConnectionsByOrganizationIDRow) Response {
	return Response{
		ID:                     connection.ID,
		OrganizationID:         connection.OrganizationID,
		ProviderID:             connection.ProviderID,
		Name:                   connection.Name,
		Status:                 connection.Status,
		OutboundAuthMethod:     connection.OutboundAuthMethod,
		AuthUsername:           connection.AuthUsername,
		HasOutboundCredentials: boolValue(connection.HasOutboundCredentials),
		InboundEnabled:         connection.InboundEnabled,
		InboundAuthMethod:      connection.InboundAuthMethod,
		InboundUsername:        connection.InboundUsername,
		HasInboundCredentials:  boolValue(connection.HasInboundCredentials),
		MaxCPS:                 connection.MaxCps,
		MaxConcurrentCalls:     connection.MaxConcurrentCalls,
		MaxDailyMinutes:        connection.MaxDailyMinutes,
		Codecs:                 connection.Codecs,
		SupportsVideo:          connection.SupportsVideo,
		SupportsFax:            connection.SupportsFax,
		CreatedAt:              pgconv.TimestamptzToTime(connection.CreatedAt),
		UpdatedAt:              pgconv.TimestamptzToTime(connection.UpdatedAt),
	}
}

func boolValue(value interface{}) bool {
	result, _ := value.(bool)
	return result
}

func sourceIPResponse(sourceIP sqlc.CarrierConnectionSourceIp) SourceIPResponse {
	return SourceIPResponse{
		ID:                  sourceIP.ID,
		OrganizationID:      sourceIP.OrganizationID,
		CarrierConnectionID: sourceIP.CarrierConnectionID,
		CIDR:                sourceIP.Cidr,
		CreatedAt:           pgconv.TimestamptzToTime(sourceIP.CreatedAt),
	}
}

func providerResponse(provider sqlc.CarrierProvider) ProviderResponse {
	return ProviderResponse{
		ID:        provider.ID,
		Slug:      provider.Slug,
		Name:      provider.Name,
		Adapter:   provider.Adapter,
		Status:    provider.Status,
		CreatedAt: pgconv.TimestamptzToTime(provider.CreatedAt),
		UpdatedAt: pgconv.TimestamptzToTime(provider.UpdatedAt),
	}
}

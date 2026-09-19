package carriers

import (
	"net/netip"
	"time"

	"github.com/google/uuid"
)

type DigestCredential struct {
	Username string `json:"username"`
	Realm    string `json:"realm"`
	Secret   string `json:"secret"`
}

type CreateRequest struct {
	ProviderID         uuid.UUID         `json:"provider_id"`
	Name               string            `json:"name"`
	Status             *string           `json:"status,omitempty"`
	OutboundCredential *DigestCredential `json:"outbound_credential,omitempty"`
	InboundEnabled     *bool             `json:"inbound_enabled,omitempty"`
	InboundAuthMethod  *string           `json:"inbound_auth_method,omitempty"`
	InboundCredential  *DigestCredential `json:"inbound_credential,omitempty"`
	MaxCPS             *int32            `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32            `json:"max_concurrent_calls,omitempty"`
	MaxDailyMinutes    *int64            `json:"max_daily_minutes,omitempty"`
	Codecs             []string          `json:"codecs,omitempty"`
	SupportsVideo      *bool             `json:"supports_video,omitempty"`
	SupportsFax        *bool             `json:"supports_fax,omitempty"`
}

type UpdateRequest struct {
	Name               *string   `json:"name,omitempty"`
	Status             *string   `json:"status,omitempty"`
	InboundEnabled     *bool     `json:"inbound_enabled,omitempty"`
	MaxCPS             *int32    `json:"max_cps,omitempty"`
	MaxConcurrentCalls *int32    `json:"max_concurrent_calls,omitempty"`
	MaxDailyMinutes    *int64    `json:"max_daily_minutes,omitempty"`
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

type Response struct {
	ID                     uuid.UUID `json:"id"`
	OrganizationID         uuid.UUID `json:"organization_id"`
	ProviderID             uuid.UUID `json:"provider_id"`
	Name                   string    `json:"name"`
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
	MaxDailyMinutes        *int64    `json:"max_daily_minutes,omitempty"`
	Codecs                 []string  `json:"codecs"`
	SupportsVideo          bool      `json:"supports_video"`
	SupportsFax            bool      `json:"supports_fax"`
	CreatedAt              time.Time `json:"created_at"`
	UpdatedAt              time.Time `json:"updated_at"`
}

type SourceIPResponse struct {
	ID                  uuid.UUID    `json:"id"`
	CarrierConnectionID uuid.UUID    `json:"carrier_connection_id"`
	CIDR                netip.Prefix `json:"cidr"`
	CreatedAt           time.Time    `json:"created_at"`
}

type ValidationResponse struct {
	Valid    bool     `json:"valid"`
	Errors   []string `json:"errors"`
	Warnings []string `json:"warnings"`
}

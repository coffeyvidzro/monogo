package numbers

import (
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type ProvisioningMode string

const (
	ProvisioningModeBYOC    ProvisioningMode = "byoc"
	ProvisioningModeManaged ProvisioningMode = "managed"
)

var (
	ErrSelectionNotFound          = errors.New("number selection not found or expired")
	ErrSelectionUnavailable       = errors.New("number selection is no longer available")
	ErrProviderRoutingUnavailable = errors.New("managed number provider routing is not configured")
)

// CreateRequest is the single public number-creation contract. BYOC callers
// provide the number identity; managed callers provide only an opaque selection.
type CreateRequest struct {
	Type                ProvisioningMode `json:"type"`
	Number              string           `json:"number,omitempty"`
	CountryCode         string           `json:"country_code,omitempty"`
	CarrierConnectionID *uuid.UUID       `json:"carrier_connection_id,omitempty"`
	SelectionID         string           `json:"selection_id,omitempty"`
	VoiceEnabled        *bool            `json:"voice_enabled,omitempty"`
	SMSEnabled          *bool            `json:"sms_enabled,omitempty"`
}

// AvailableSearchRequest is the provider-neutral customer search contract.
type AvailableSearchRequest struct {
	CountryCode string
	Contains    string
}

type MoneyQuote struct {
	AmountMinor int64  `json:"amount_minor"`
	Currency    string `json:"currency"`
}

// AvailableNumberResponse exposes an opaque, short-lived selection handle and
// the Leamout-owned customer price. Provider inventory IDs, SKUs, and wholesale
// economics remain internal.
type AvailableNumberResponse struct {
	SelectionID  string     `json:"selection_id"`
	Number       string     `json:"number"`
	CountryCode  string     `json:"country_code"`
	VoiceEnabled bool       `json:"voice_enabled"`
	Price        MoneyQuote `json:"price"`
}

// ManagedNumberCandidate retains provider purchase inputs and the server-owned
// quote behind selection_id. Provider adapters populate only provider fields;
// Commercial attaches the customer price before the selection is persisted.
type ManagedNumberCandidate struct {
	Provider              string
	ProviderInventoryID   string
	ProviderProductID     string
	Number                string
	CountryCode           string
	ChannelsIncludedCount int
	PriceID               uuid.UUID
	PriceAmountMinor      int64
	PriceCurrency         string
}

// ManagedNumberPurchaseAuthorization is durable proof that customer funds were
// reserved before the provider operation became runnable.
type ManagedNumberPurchaseAuthorization struct {
	ID            uuid.UUID `json:"id"`
	ReservationID uuid.UUID `json:"reservation_id"`
	PriceID       uuid.UUID `json:"price_id"`
	AmountMinor   int64     `json:"amount_minor"`
	Currency      string    `json:"currency"`
}

type ProviderOperationRequest struct {
	Provider                  string                             `json:"provider"`
	ProviderInventoryID       string                             `json:"available_did_id"`
	ProviderProductID         string                             `json:"sku_id"`
	Number                    string                             `json:"number"`
	CountryCode               string                             `json:"country_code"`
	CarrierConnectionID       uuid.UUID                          `json:"carrier_connection_id"`
	ProviderRoutingResourceID string                             `json:"provider_routing_resource_id"`
	PurchaseAuthorization     ManagedNumberPurchaseAuthorization `json:"purchase_authorization"`
}

type ProviderOrderRequest struct {
	ProviderInventoryID string
	ProviderProductID   string
	ExternalReferenceID string
}

type ProviderOrder struct {
	ID     string
	Status string
}

type ProviderNumber struct {
	ID                string
	Number            string
	RoutingResourceID string
}

type UpdateRequest struct {
	VoiceEnabled *bool `json:"voice_enabled,omitempty"`
	SMSEnabled   *bool `json:"sms_enabled,omitempty"`
}

type CarrierConnectionRequest struct {
	CarrierConnectionID uuid.UUID `json:"carrier_connection_id"`
}

type NumberError struct {
	Code    string `json:"code,omitempty"`
	Message string `json:"message"`
}

type Response struct {
	ID                  uuid.UUID        `json:"id"`
	OrganizationID      uuid.UUID        `json:"organization_id"`
	Type                ProvisioningMode `json:"type"`
	Number              string           `json:"number"`
	CountryCode         string           `json:"country_code"`
	CarrierConnectionID *uuid.UUID       `json:"carrier_connection_id,omitempty"`
	VoiceEnabled        bool             `json:"voice_enabled"`
	SMSEnabled          bool             `json:"sms_enabled"`
	Status              string           `json:"status"`
	Error               *NumberError     `json:"error,omitempty"`
	CreatedAt           time.Time        `json:"created_at"`
	UpdatedAt           time.Time        `json:"updated_at"`
}

func response(number sqlc.PhoneNumber) Response {
	result := Response{
		ID:             number.ID,
		OrganizationID: number.OrganizationID,
		Type:           ProvisioningMode(number.ProvisioningMode),
		Number:         number.Number,
		CountryCode:    number.CountryCode,
		VoiceEnabled:   number.VoiceEnabled,
		SMSEnabled:     number.SmsEnabled,
		Status:         number.Status,
		CreatedAt:      pgconv.TimestamptzToTime(number.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(number.UpdatedAt),
	}
	if number.ErrorMessage != nil {
		result.Error = &NumberError{Message: *number.ErrorMessage}
		if number.ErrorCode != nil {
			result.Error.Code = *number.ErrorCode
		}
	}

	// Managed carrier bindings are Leamout implementation details.
	if number.ProvisioningMode == string(ProvisioningModeBYOC) {
		result.CarrierConnectionID = number.CarrierConnectionID
	}
	return result
}

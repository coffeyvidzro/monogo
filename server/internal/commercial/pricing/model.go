package pricing

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"

	ProductSMSOutbound      = "sms_outbound"
	ProductWhatsAppOutbound = "whatsapp_outbound"
	ProductNumberPurchase   = "number_purchase"
	ProductNumberRenewal    = "number_renewal"
)

var (
	ErrInvalidInput      = errors.New("invalid pricing input")
	ErrVoiceRateNotFound = errors.New("voice rate not found")
	ErrVoiceRateConflict = errors.New("voice rate conflict")
)

type VoiceRate struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    *uuid.UUID `json:"organization_id,omitempty"`
	DestinationPrefix string     `json:"destination_prefix"`
	Direction         string     `json:"direction"`
	Currency          string     `json:"currency"`
	RateMicros        int64      `json:"rate_micros"` // USD micros per started 60-second managed voice minute.
	EffectiveAt       time.Time  `json:"effective_at"`
	ExpiresAt         *time.Time `json:"expires_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
}

type CreateVoiceRateRequest struct {
	OrganizationID    *uuid.UUID
	DestinationPrefix string
	Direction         string
	Currency          string
	RateMicros        int64
	EffectiveAt       time.Time
	ExpiresAt         *time.Time
}

type ResolveVoiceRateRequest struct {
	OrganizationID    uuid.UUID
	DestinationDigits string
	Direction         string
	Currency          string
	ResolvedAt        time.Time
}

type ProductRate struct {
	ID             uuid.UUID
	OrganizationID *uuid.UUID
	Product        string
	Selector       string
	Currency       string
	RateMicros     int64
	EffectiveAt    time.Time
	ExpiresAt      *time.Time
	CreatedAt      time.Time
}

type ResolveProductRateRequest struct {
	OrganizationID uuid.UUID
	Product        string
	Selector       string
	Currency       string
	ResolvedAt     time.Time
}

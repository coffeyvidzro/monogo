package numbers

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

// AvailableNumber is display-only inventory. It is not a purchase offer or
// authorization to order an upstream DID.
type AvailableNumber struct {
	Number string `json:"number"`
}

type ManagedPurchaseRequest struct {
	Number      string `json:"number"`
	CountryCode string `json:"country_code"`
}

type ManagedOrder struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	PhoneNumberID       *uuid.UUID `json:"phone_number_id,omitempty"`
	Number              string     `json:"number"`
	CountryCode         string     `json:"country_code"`
	Status              string     `json:"status"`
	SubmittedAt         *time.Time `json:"submitted_at,omitempty"`
	OwnershipVerifiedAt *time.Time `json:"ownership_verified_at,omitempty"`
	RoutingVerifiedAt   *time.Time `json:"routing_verified_at,omitempty"`
	ActivatedAt         *time.Time `json:"activated_at,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
	ProviderID          uuid.UUID  `json:"-"`
	AvailableDIDID      string     `json:"-"`
	SKUID               string     `json:"-"`
	ProviderOrderID     *string    `json:"-"`
	ProviderDIDID       *string    `json:"-"`
	InboundTrunkID      *string    `json:"-"`
	IdempotencyKey      string     `json:"-"`
	RequestHash         string     `json:"-"`
	ReconcileAfter      time.Time  `json:"-"`
	ReconcileAttempts   int32      `json:"-"`
}

func managedOrder(row sqlc.ManagedNumberOrder) ManagedOrder {
	return ManagedOrder{
		ID: row.ID, OrganizationID: row.OrganizationID, ProviderID: row.ProviderID,
		PhoneNumberID: row.PhoneNumberID, IdempotencyKey: row.IdempotencyKey,
		RequestHash: row.RequestHash, Number: row.Number, CountryCode: row.CountryCode,
		AvailableDIDID: row.AvailableDidID, SKUID: row.SkuID,
		ProviderOrderID: row.ProviderOrderID, ProviderDIDID: row.ProviderDidID,
		InboundTrunkID: row.InboundTrunkID, Status: row.Status,
		SubmittedAt:         pgconv.TimestamptzToTimePtr(row.SubmittedAt),
		OwnershipVerifiedAt: pgconv.TimestamptzToTimePtr(row.OwnershipVerifiedAt),
		RoutingVerifiedAt:   pgconv.TimestamptzToTimePtr(row.RoutingVerifiedAt),
		ActivatedAt:         pgconv.TimestamptzToTimePtr(row.ActivatedAt),
		ReconcileAfter:      pgconv.TimestamptzToTime(row.ReconcileAfter),
		ReconcileAttempts:   row.ReconcileAttempts,
		CreatedAt:           pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

type CreateBYOCRequest struct {
	Number              string     `json:"number"`
	CountryCode         string     `json:"country_code"`
	CarrierConnectionID *uuid.UUID `json:"carrier_connection_id,omitempty"`
	VoiceEnabled        *bool      `json:"voice_enabled,omitempty"`
	SmsEnabled          *bool      `json:"sms_enabled,omitempty"`
}

type UpdateRequest struct {
	VoiceEnabled *bool `json:"voice_enabled,omitempty"`
	SmsEnabled   *bool `json:"sms_enabled,omitempty"`
}

type SetCarrierConnectionRequest struct {
	CarrierConnectionID uuid.UUID `json:"carrier_connection_id"`
}

type Response struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	Number              string     `json:"number"`
	CountryCode         string     `json:"country_code"`
	Type                string     `json:"type"`
	CarrierConnectionID *uuid.UUID `json:"carrier_connection_id,omitempty"`
	VoiceEnabled        bool       `json:"voice_enabled"`
	SmsEnabled          bool       `json:"sms_enabled"`
	Status              string     `json:"status"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func response(row sqlc.PhoneNumber) Response {
	// A managed carrier connection belongs to Leamout. It identifies an
	// upstream route and must not become part of the customer API contract.
	carrierConnectionID := row.CarrierConnectionID
	if row.ProvisioningMode == "managed" {
		carrierConnectionID = nil
	}
	return Response{
		ID:                  row.ID,
		OrganizationID:      row.OrganizationID,
		Number:              row.Number,
		CountryCode:         row.CountryCode,
		Type:                row.ProvisioningMode,
		CarrierConnectionID: carrierConnectionID,
		VoiceEnabled:        row.VoiceEnabled,
		SmsEnabled:          row.SmsEnabled,
		Status:              row.Status,
		CreatedAt:           pgconv.TimestamptzToTime(row.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(row.UpdatedAt),
	}
}

package numbers

import (
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

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

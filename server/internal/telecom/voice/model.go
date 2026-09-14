package voice

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type ApplicationResponse struct {
	ID                 uuid.UUID `json:"id"`
	OrganizationID     uuid.UUID `json:"organization_id"`
	Name               string    `json:"name"`
	RingTimeoutSeconds int32     `json:"ring_timeout_seconds"`
	CallerID           *string   `json:"caller_id,omitempty"`
	Status             string    `json:"status"`
	VoiceURL           *string   `json:"voice_url,omitempty"`
	CallbackURL        *string   `json:"callback_url,omitempty"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type BindingResponse struct {
	ID                 uuid.UUID  `json:"id"`
	VoiceApplicationID uuid.UUID  `json:"voice_application_id"`
	PhoneNumberID      *uuid.UUID `json:"phone_number_id,omitempty"`
	SIPDomainID        *uuid.UUID `json:"sip_domain_id,omitempty"`
	SubscriberID       *uuid.UUID `json:"subscriber_id,omitempty"`
	CreatedAt          time.Time  `json:"created_at"`
}

type CreateApplicationRequest struct {
	Name               string  `json:"name"`
	RingTimeoutSeconds *int32  `json:"ring_timeout_seconds,omitempty"`
	CallerID           *string `json:"caller_id,omitempty"`
	VoiceURL           *string `json:"voice_url,omitempty"`
	CallbackURL        *string `json:"callback_url,omitempty"`
}

type UpdateApplicationRequest struct {
	Name               *string `json:"name,omitempty"`
	RingTimeoutSeconds *int32  `json:"ring_timeout_seconds,omitempty"`
	CallerID           *string `json:"caller_id,omitempty"`
	VoiceURL           *string `json:"voice_url,omitempty"`
	CallbackURL        *string `json:"callback_url,omitempty"`
}

type CreateBindingRequest struct {
	PhoneNumberID *uuid.UUID `json:"phone_number_id,omitempty"`
	SIPDomainID   *uuid.UUID `json:"sip_domain_id,omitempty"`
	SubscriberID  *uuid.UUID `json:"subscriber_id,omitempty"`
}

func applicationResponse(app sqlc.VoiceApplication) ApplicationResponse {
	return ApplicationResponse{
		ID:                 app.ID,
		OrganizationID:     app.OrganizationID,
		Name:               app.Name,
		RingTimeoutSeconds: app.RingTimeoutSeconds,
		CallerID:           app.CallerID,
		Status:             app.Status,
		VoiceURL:           app.VoiceUrl,
		CallbackURL:        app.CallbackUrl,
		CreatedAt:          pgconv.TimestamptzToTime(app.CreatedAt),
		UpdatedAt:          pgconv.TimestamptzToTime(app.UpdatedAt),
	}
}

func bindingResponse(binding sqlc.VoiceBinding) BindingResponse {
	return BindingResponse{
		ID:                 binding.ID,
		VoiceApplicationID: binding.VoiceApplicationID,
		PhoneNumberID:      binding.PhoneNumberID,
		SIPDomainID:        binding.SipDomainID,
		SubscriberID:       binding.SubscriberID,
		CreatedAt:          pgconv.TimestamptzToTime(binding.CreatedAt),
	}
}

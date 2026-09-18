package calls

import (
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

var ErrChannelUnavailable = errors.New("active call channel unavailable")

type Direction string

const (
	DirectionInbound  Direction = "inbound"
	DirectionOutbound Direction = "outbound"
)

type State string

const (
	StateInitiating State = "initiating"
	StateRinging    State = "ringing"
	StateAnswered   State = "answered"
	StateActive     State = "active"
	StateCompleted  State = "completed"
	StateFailed     State = "failed"
	StateCancelled  State = "cancelled"
)

type MediaState string

const (
	MediaStateActive MediaState = "active"
	MediaStateHeld   MediaState = "held"
)

type LifecycleEventType string

const (
	LifecycleInitiated LifecycleEventType = "initiated"
	LifecycleRinging   LifecycleEventType = "ringing"
	LifecycleAnswered  LifecycleEventType = "answered"
	LifecycleActive    LifecycleEventType = "active"
	LifecycleHeld      LifecycleEventType = "held"
	LifecycleResumed   LifecycleEventType = "resumed"
	LifecycleCompleted LifecycleEventType = "completed"
	LifecycleFailed    LifecycleEventType = "failed"
	LifecycleCancelled LifecycleEventType = "cancelled"
)

type LifecycleEvent struct {
	CallID       uuid.UUID
	ChannelID    string
	Type         LifecycleEventType
	OccurredAt   time.Time
	HangupReason *string
}

type InboundAdmissionRequest struct {
	ChannelID            string
	SIPCallID            string
	OrganizationID       uuid.UUID
	ApplicationID        uuid.UUID
	PhoneNumberID        uuid.UUID
	VoiceBindingID       uuid.UUID
	CarrierConnectionID  uuid.UUID
	FromURI              string
	ToURI                string
	OccurredAt           time.Time
}

type CreateRequest struct {
	ApplicationID   *uuid.UUID `json:"application_id,omitempty"`
	TrunkID         *uuid.UUID `json:"trunk_id,omitempty"`
	FromURI         string     `json:"from_uri"`
	ToURI           string     `json:"to_uri"`
	Privacy         bool       `json:"privacy,omitempty"`
	DTMFMode        string     `json:"dtmf_mode,omitempty"`
	MediaEncryption string     `json:"media_encryption,omitempty"`

	Direction Direction `json:"-"`
	SIPCallID *string   `json:"-"`
}

type TransferActionRequest struct {
	Destination string `json:"destination"`
}

type PlayActionRequest struct {
	Path string `json:"path"`
}

type RecordActionRequest struct {
	Path   string `json:"path"`
	Action string `json:"action,omitempty"`
}

type DTMFActionRequest struct {
	Digits string `json:"digits"`
}

type ListRequest struct {
	State  *string
	Offset int32
	Limit  int32
}

type RouteAttribution struct {
	CarrierConnectionID *uuid.UUID
	TrunkID             *uuid.UUID
	TrunkEndpointID     *uuid.UUID
}

type CallResponse struct {
	ID                  uuid.UUID  `json:"id"`
	OrganizationID      uuid.UUID  `json:"organization_id"`
	ApplicationID       *uuid.UUID `json:"application_id,omitempty"`
	CarrierConnectionID *uuid.UUID `json:"carrier_connection_id,omitempty"`
	TrunkID             *uuid.UUID `json:"trunk_id,omitempty"`
	TrunkEndpointID     *uuid.UUID `json:"trunk_endpoint_id,omitempty"`
	Direction           string     `json:"direction"`
	State               string     `json:"state"`
	MediaState          string     `json:"media_state"`
	FromURI             string     `json:"from_uri"`
	ToURI               string     `json:"to_uri"`
	SIPCallID           *string    `json:"sip_call_id,omitempty"`
	ProviderID          *uuid.UUID `json:"provider_id,omitempty"`
	StartedAt           *time.Time `json:"started_at,omitempty"`
	AnsweredAt          *time.Time `json:"answered_at,omitempty"`
	EndedAt             *time.Time `json:"ended_at,omitempty"`
	HangupReason        *string    `json:"hangup_reason,omitempty"`
	CreatedAt           time.Time  `json:"created_at"`
	UpdatedAt           time.Time  `json:"updated_at"`
}

func callResponse(call sqlc.Call) CallResponse {
	return CallResponse{
		ID:                  call.ID,
		OrganizationID:      call.OrganizationID,
		ApplicationID:       call.ApplicationID,
		CarrierConnectionID: call.CarrierConnectionID,
		TrunkID:             call.TrunkID,
		TrunkEndpointID:     call.TrunkEndpointID,
		Direction:           call.Direction,
		State:               call.State,
		MediaState:          call.MediaState,
		FromURI:             call.FromUri,
		ToURI:               call.ToUri,
		SIPCallID:           call.SipCallID,
		ProviderID:          call.ProviderID,
		StartedAt:           pgconv.TimestamptzToTimePtr(call.StartedAt),
		AnsweredAt:          pgconv.TimestamptzToTimePtr(call.AnsweredAt),
		EndedAt:             pgconv.TimestamptzToTimePtr(call.EndedAt),
		HangupReason:        call.HangupReason,
		CreatedAt:           pgconv.TimestamptzToTime(call.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(call.UpdatedAt),
	}
}

package calls

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

// CreateCallRequest contains the public routing intent for an outbound call.
type CreateCallRequest struct {
	ApplicationID   *uuid.UUID        `json:"application_id,omitempty"`
	TrunkID         *uuid.UUID        `json:"trunk_id,omitempty"`
	From            string            `json:"from"`
	To              string            `json:"to"`
	Variables       map[string]string `json:"variables,omitempty"`
	Privacy         bool              `json:"privacy,omitempty"`
	DTMFMode        string            `json:"dtmf_mode,omitempty"`
	Codecs          []string          `json:"codecs,omitempty"`
	MediaEncryption string            `json:"media_encryption,omitempty"`
}

// RouteAttribution records the concrete route selected for an outbound call.
type RouteAttribution struct {
	CarrierConnectionID uuid.UUID
	TrunkID             uuid.UUID
	TrunkEndpointID     uuid.UUID
}

// OriginateRequest is the internal media-controller command produced after
// Leamout resolves a public call request to a concrete telecom route.
type OriginateRequest struct {
	CarrierConnectionID uuid.UUID
	Host                string
	Port                int32
	Transport           string
	Destination         string
	CallerID            string
	Variables           map[string]string
	Privacy             bool
	DTMFMode            string
	Codecs              []string
	MediaEncryption     string
}

type InboundCallEvent struct {
	OrganizationID      uuid.UUID
	CarrierConnectionID uuid.UUID
	PhoneNumberID       uuid.UUID
	ApplicationID       uuid.UUID
	ChannelID           string
	From                string
	To                  string
	HangupCause         string
	WasAnswered         bool
}

type TransferRequest struct {
	Destination string `json:"destination"`
	Dialplan    string `json:"dialplan,omitempty"`
	Context     string `json:"context,omitempty"`
}

type PlayRequest struct {
	Path string `json:"path"`
}

type RecordRequest struct {
	Action string `json:"action"`
	Path   string `json:"path"`
}

type DTMFRequest struct {
	Digits string `json:"digits"`
}

type CallResponse struct {
	ID                  uuid.UUID      `json:"id"`
	OrganizationID      uuid.UUID      `json:"organization_id"`
	ApplicationID       *uuid.UUID     `json:"application_id,omitempty"`
	CarrierConnectionID *uuid.UUID     `json:"carrier_connection_id,omitempty"`
	TrunkID             *uuid.UUID     `json:"trunk_id,omitempty"`
	TrunkEndpointID     *uuid.UUID     `json:"trunk_endpoint_id,omitempty"`
	Direction           string         `json:"direction"`
	State               string         `json:"state"`
	MediaState          CallMediaState `json:"media_state"`
	From                string         `json:"from"`
	To                  string         `json:"to"`
	SIPCallID           *string        `json:"sip_call_id,omitempty"`
	StartedAt           *time.Time     `json:"started_at,omitempty"`
	AnsweredAt          *time.Time     `json:"answered_at,omitempty"`
	EndedAt             *time.Time     `json:"ended_at,omitempty"`
	HangupReason        *string        `json:"hangup_reason,omitempty"`
	CreatedAt           time.Time      `json:"created_at"`
	UpdatedAt           time.Time      `json:"updated_at"`
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
		MediaState:          CallMediaState(call.MediaState),
		From:                call.FromUri,
		To:                  call.ToUri,
		SIPCallID:           call.SipCallID,
		StartedAt:           pgconv.TimestamptzToTimePtr(call.StartedAt),
		AnsweredAt:          pgconv.TimestamptzToTimePtr(call.AnsweredAt),
		EndedAt:             pgconv.TimestamptzToTimePtr(call.EndedAt),
		HangupReason:        call.HangupReason,
		CreatedAt:           pgconv.TimestamptzToTime(call.CreatedAt),
		UpdatedAt:           pgconv.TimestamptzToTime(call.UpdatedAt),
	}
}

type CallEvent struct {
	EventType      CallEventType  `json:"event_type"`
	CallID         string         `json:"call_id"`
	SIPCallID      string         `json:"sip_call_id,omitempty"`
	ApplicationID  string         `json:"application_id"`
	OrganizationID string         `json:"organization_id"`
	From           string         `json:"from"`
	To             string         `json:"to"`
	Direction      CallDirection  `json:"direction"`
	Status         CallStatus     `json:"status"`
	MediaState     CallMediaState `json:"media_state"`
	HangupReason   string         `json:"hangup_reason,omitempty"`
	DurationSec    int            `json:"duration_seconds,omitempty"`
	OccurredAt     time.Time      `json:"occurred_at"`
	RecordingURL   string         `json:"recording_url,omitempty"`
}

type CallEventType string

const (
	EventCallInitiated CallEventType = "call.initiated"
	EventCallRinging   CallEventType = "call.ringing"
	EventCallAnswered  CallEventType = "call.answered"
	EventCallActive    CallEventType = "call.active"
	EventCallHeld      CallEventType = "call.held"
	EventCallResumed   CallEventType = "call.resumed"
	EventCallCompleted CallEventType = "call.completed"
	EventCallFailed    CallEventType = "call.failed"
	EventCallCancelled CallEventType = "call.cancelled"
	EventCallRecording CallEventType = "call.recording.available"
)

type CallDirection string

const (
	DirectionInbound  CallDirection = "inbound"
	DirectionOutbound CallDirection = "outbound"
)

// CallState is the persisted lifecycle state stored in calls.state.
type CallState string

const (
	StateInitiating CallState = "initiating"
	StateRinging    CallState = "ringing"
	StateAnswered   CallState = "answered"
	StateActive     CallState = "active"
	StateCompleted  CallState = "completed"
	StateFailed     CallState = "failed"
	StateCancelled  CallState = "cancelled"
)

// CallMediaState is the persisted media condition stored in calls.media_state.
type CallMediaState string

const (
	MediaStateActive CallMediaState = "active"
	MediaStateHeld   CallMediaState = "held"
)

// CallStatus is the public status vocabulary used in emitted call events.
type CallStatus string

const (
	StatusInitiated CallStatus = "initiated"
	StatusRinging   CallStatus = "ringing"
	StatusAnswered  CallStatus = "answered"
	StatusActive    CallStatus = "active"
	StatusCompleted CallStatus = "completed"
	StatusFailed    CallStatus = "failed"
	StatusCancelled CallStatus = "cancelled"
	StatusBusy      CallStatus = "busy"
	StatusNoAnswer  CallStatus = "no-answer"
)

func statusForState(state string) CallStatus {
	switch CallState(state) {
	case StateInitiating:
		return StatusInitiated
	case StateRinging:
		return StatusRinging
	case StateAnswered:
		return StatusAnswered
	case StateActive:
		return StatusActive
	case StateCompleted:
		return StatusCompleted
	case StateFailed:
		return StatusFailed
	case StateCancelled:
		return StatusCancelled
	default:
		return CallStatus(state)
	}
}

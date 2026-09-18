package calling

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/google/uuid"
)

var (
	ErrUnsupportedEvent    = errors.New("unsupported FreeSWITCH call event")
	ErrUncorrelatedEvent   = errors.New("uncorrelated FreeSWITCH call event")
	ErrNotInboundAdmission = errors.New("not an inbound FreeSWITCH admission event")
)

type LifecycleType string

const (
	LifecycleInitiated LifecycleType = "initiated"
	LifecycleRinging   LifecycleType = "ringing"
	LifecycleAnswered  LifecycleType = "answered"
	LifecycleActive    LifecycleType = "active"
	LifecycleHeld      LifecycleType = "held"
	LifecycleResumed   LifecycleType = "resumed"
	LifecycleCompleted LifecycleType = "completed"
	LifecycleFailed    LifecycleType = "failed"
	LifecycleCancelled LifecycleType = "cancelled"
)

type LifecycleEvent struct {
	CallID       uuid.UUID
	ChannelID    string
	Type         LifecycleType
	OccurredAt   time.Time
	HangupReason *string
}

type InboundEvent struct {
	ChannelID           string
	SIPCallID           string
	OrganizationID      uuid.UUID
	ApplicationID       uuid.UUID
	PhoneNumberID       uuid.UUID
	VoiceBindingID      uuid.UUID
	CarrierConnectionID uuid.UUID
	FromURI             string
	ToURI               string
	OccurredAt          time.Time
}

func FreeSWITCHEvents() []string {
	return []string{
		"CHANNEL_CREATE",
		"CHANNEL_ANSWER",
		"CHANNEL_HOLD",
		"CHANNEL_UNHOLD",
		"CHANNEL_HANGUP_COMPLETE",
	}
}

func TranslateInboundFreeSWITCHEvent(event freeswitch.Event) (InboundEvent, error) {
	if event.Name != "CHANNEL_CREATE" {
		return InboundEvent{}, fmt.Errorf("%w: %s", ErrNotInboundAdmission, event.Name)
	}
	if strings.TrimSpace(event.Header("variable_leamout_call_id")) != "" {
		return InboundEvent{}, ErrNotInboundAdmission
	}

	headers := map[string]string{
		"organization_id":        event.Header("variable_sip_h_X-Leamout-Organization-ID"),
		"carrier_connection_id": event.Header("variable_sip_h_X-Leamout-Carrier-Connection-ID"),
		"phone_number_id":       event.Header("variable_sip_h_X-Leamout-Phone-Number-ID"),
		"voice_binding_id":      event.Header("variable_sip_h_X-Leamout-Voice-Binding-ID"),
		"application_id":        event.Header("variable_sip_h_X-Leamout-Voice-Application-ID"),
	}
	hasTrustedMetadata := false
	for _, value := range headers {
		if strings.TrimSpace(value) != "" {
			hasTrustedMetadata = true
			break
		}
	}
	if !hasTrustedMetadata {
		return InboundEvent{}, ErrNotInboundAdmission
	}

	parseID := func(name string) (uuid.UUID, error) {
		id, err := uuid.Parse(strings.TrimSpace(headers[name]))
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid inbound %s", name)
		}
		return id, nil
	}

	organizationID, err := parseID("organization_id")
	if err != nil { return InboundEvent{}, err }
	carrierConnectionID, err := parseID("carrier_connection_id")
	if err != nil { return InboundEvent{}, err }
	phoneNumberID, err := parseID("phone_number_id")
	if err != nil { return InboundEvent{}, err }
	voiceBindingID, err := parseID("voice_binding_id")
	if err != nil { return InboundEvent{}, err }
	applicationID, err := parseID("application_id")
	if err != nil { return InboundEvent{}, err }

	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return InboundEvent{}, fmt.Errorf("FreeSWITCH inbound event is missing Unique-ID")
	}
	sipCallID := strings.TrimSpace(event.Header("variable_sip_call_id"))
	if sipCallID == "" {
		return InboundEvent{}, fmt.Errorf("FreeSWITCH inbound event is missing SIP Call-ID")
	}
	fromURI := firstNonEmpty(event.Header("variable_sip_from_uri"), event.Header("Caller-Caller-ID-Number"), event.Header("Caller-ANI"))
	if fromURI == "" {
		return InboundEvent{}, fmt.Errorf("FreeSWITCH inbound event is missing caller identity")
	}
	toURI := firstNonEmpty(event.Header("Caller-Destination-Number"), event.Header("variable_sip_to_user"))
	if toURI == "" {
		return InboundEvent{}, fmt.Errorf("FreeSWITCH inbound event is missing called number")
	}
	occurredAt, err := freeSWITCHEventTime(event)
	if err != nil {
		return InboundEvent{}, err
	}

	return InboundEvent{
		ChannelID: channelID, SIPCallID: sipCallID, OrganizationID: organizationID,
		ApplicationID: applicationID, PhoneNumberID: phoneNumberID, VoiceBindingID: voiceBindingID,
		CarrierConnectionID: carrierConnectionID, FromURI: strings.TrimSpace(fromURI),
		ToURI: strings.TrimSpace(toURI), OccurredAt: occurredAt,
	}, nil
}

func TranslateFreeSWITCHEvent(event freeswitch.Event) (LifecycleEvent, error) {
	eventType, err := lifecycleEventType(event)
	if err != nil {
		return LifecycleEvent{}, err
	}
	callID, err := uuid.Parse(strings.TrimSpace(event.Header("variable_leamout_call_id")))
	if err != nil {
		return LifecycleEvent{}, fmt.Errorf("%w: missing valid Leamout call id", ErrUncorrelatedEvent)
	}
	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return LifecycleEvent{}, fmt.Errorf("FreeSWITCH call event is missing Unique-ID")
	}
	occurredAt, err := freeSWITCHEventTime(event)
	if err != nil {
		return LifecycleEvent{}, err
	}

	result := LifecycleEvent{CallID: callID, ChannelID: channelID, Type: eventType, OccurredAt: occurredAt}
	if event.Name == "CHANNEL_HANGUP_COMPLETE" {
		cause := strings.TrimSpace(firstNonEmpty(event.Header("Hangup-Cause"), event.Header("variable_hangup_cause")))
		if cause != "" {
			result.HangupReason = &cause
		}
	}
	return result, nil
}

func lifecycleEventType(event freeswitch.Event) (LifecycleType, error) {
	switch event.Name {
	case "CHANNEL_CREATE":
		return LifecycleInitiated, nil
	case "CHANNEL_ANSWER":
		return LifecycleAnswered, nil
	case "CHANNEL_HOLD":
		return LifecycleHeld, nil
	case "CHANNEL_UNHOLD":
		return LifecycleResumed, nil
	case "CHANNEL_HANGUP_COMPLETE":
		if eventAnswered(event) { return LifecycleCompleted, nil }
		if strings.EqualFold(strings.TrimSpace(event.Header("Hangup-Cause")), "ORIGINATOR_CANCEL") {
			return LifecycleCancelled, nil
		}
		return LifecycleFailed, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEvent, event.Name)
	}
}

func eventAnswered(event freeswitch.Event) bool {
	for _, value := range []string{event.Header("Answered"), event.Header("variable_answered")} {
		if strings.EqualFold(strings.TrimSpace(value), "true") { return true }
	}
	for _, value := range []string{event.Header("variable_answer_epoch"), event.Header("answer_epoch"), event.Header("billmsec"), event.Header("variable_billmsec")} {
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil && parsed > 0 { return true }
	}
	return false
}

func freeSWITCHEventTime(event freeswitch.Event) (time.Time, error) {
	raw := strings.TrimSpace(event.Header("Event-Date-Timestamp"))
	if raw == "" { return time.Now().UTC(), nil }
	micros, err := strconv.ParseInt(raw, 10, 64)
	if err != nil { return time.Time{}, fmt.Errorf("parse FreeSWITCH event timestamp: %w", err) }
	return time.UnixMicro(micros).UTC(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" { return value }
	}
	return ""
}

package calls

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
	ErrUnsupportedEvent     = errors.New("unsupported FreeSWITCH call event")
	ErrUncorrelatedEvent    = errors.New("uncorrelated FreeSWITCH call event")
	ErrNotInboundAdmission  = errors.New("not an inbound FreeSWITCH admission event")
)


// TranslateInboundFreeSWITCHEvent converts the first trusted carrier-ingress
// channel event into the admission request used to create a Leamout call.
// External SIP peers cannot supply these values directly to FreeSWITCH:
// OpenSIPS strips them and writes its own resolved X-Leamout metadata.
func TranslateInboundFreeSWITCHEvent(event freeswitch.Event) (InboundAdmissionRequest, error) {
	if event.Name != "CHANNEL_CREATE" {
		return InboundAdmissionRequest{}, fmt.Errorf("%w: %s", ErrNotInboundAdmission, event.Name)
	}
	if strings.TrimSpace(event.Header("variable_leamout_call_id")) != "" {
		return InboundAdmissionRequest{}, ErrNotInboundAdmission
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
		return InboundAdmissionRequest{}, ErrNotInboundAdmission
	}

	parseID := func(name string) (uuid.UUID, error) {
		value := strings.TrimSpace(headers[name])
		id, err := uuid.Parse(value)
		if err != nil {
			return uuid.Nil, fmt.Errorf("invalid inbound %s", name)
		}
		return id, nil
	}

	organizationID, err := parseID("organization_id")
	if err != nil {
		return InboundAdmissionRequest{}, err
	}
	carrierConnectionID, err := parseID("carrier_connection_id")
	if err != nil {
		return InboundAdmissionRequest{}, err
	}
	phoneNumberID, err := parseID("phone_number_id")
	if err != nil {
		return InboundAdmissionRequest{}, err
	}
	voiceBindingID, err := parseID("voice_binding_id")
	if err != nil {
		return InboundAdmissionRequest{}, err
	}
	applicationID, err := parseID("application_id")
	if err != nil {
		return InboundAdmissionRequest{}, err
	}

	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return InboundAdmissionRequest{}, fmt.Errorf("FreeSWITCH inbound event is missing Unique-ID")
	}

	sipCallID := strings.TrimSpace(event.Header("variable_sip_call_id"))
	if sipCallID == "" {
		return InboundAdmissionRequest{}, fmt.Errorf("FreeSWITCH inbound event is missing SIP Call-ID")
	}

	fromURI := firstNonEmpty(
		event.Header("variable_sip_from_uri"),
		event.Header("Caller-Caller-ID-Number"),
		event.Header("Caller-ANI"),
	)
	if fromURI == "" {
		return InboundAdmissionRequest{}, fmt.Errorf("FreeSWITCH inbound event is missing caller identity")
	}

	toURI := firstNonEmpty(
		event.Header("Caller-Destination-Number"),
		event.Header("variable_sip_to_user"),
	)
	if toURI == "" {
		return InboundAdmissionRequest{}, fmt.Errorf("FreeSWITCH inbound event is missing called number")
	}

	occurredAt, err := freeSWITCHEventTime(event)
	if err != nil {
		return InboundAdmissionRequest{}, err
	}

	return InboundAdmissionRequest{
		ChannelID:           channelID,
		SIPCallID:           sipCallID,
		OrganizationID:      organizationID,
		ApplicationID:       applicationID,
		PhoneNumberID:       phoneNumberID,
		VoiceBindingID:      voiceBindingID,
		CarrierConnectionID: carrierConnectionID,
		FromURI:             strings.TrimSpace(fromURI),
		ToURI:               strings.TrimSpace(toURI),
		OccurredAt:          occurredAt,
	}, nil
}

// TranslateFreeSWITCHEvent converts a raw FreeSWITCH channel event into a
// normalized call lifecycle event. Leamout call identity is carried explicitly
// in variable_leamout_call_id; FreeSWITCH Unique-ID remains the channel ID.
func TranslateFreeSWITCHEvent(event freeswitch.Event) (LifecycleEvent, error) {
	eventType, err := lifecycleEventType(event)
	if err != nil {
		return LifecycleEvent{}, err
	}

	rawCallID := strings.TrimSpace(event.Header("variable_leamout_call_id"))
	callID, err := uuid.Parse(rawCallID)
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

	result := LifecycleEvent{
		CallID:     callID,
		ChannelID:  channelID,
		Type:       eventType,
		OccurredAt: occurredAt,
	}

	if event.Name == "CHANNEL_HANGUP_COMPLETE" {
		cause := strings.TrimSpace(firstNonEmpty(
			event.Header("Hangup-Cause"),
			event.Header("variable_hangup_cause"),
		))
		if cause != "" {
			result.HangupReason = &cause
		}
	}

	return result, nil
}

func lifecycleEventType(event freeswitch.Event) (LifecycleEventType, error) {
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
		if eventAnswered(event) {
			return LifecycleCompleted, nil
		}
		if strings.EqualFold(strings.TrimSpace(event.Header("Hangup-Cause")), "ORIGINATOR_CANCEL") {
			return LifecycleCancelled, nil
		}
		return LifecycleFailed, nil
	default:
		return "", fmt.Errorf("%w: %s", ErrUnsupportedEvent, event.Name)
	}
}

func eventAnswered(event freeswitch.Event) bool {
	for _, value := range []string{
		event.Header("Answered"),
		event.Header("variable_answered"),
	} {
		if strings.EqualFold(strings.TrimSpace(value), "true") {
			return true
		}
	}

	for _, value := range []string{
		event.Header("variable_answer_epoch"),
		event.Header("answer_epoch"),
		event.Header("billmsec"),
		event.Header("variable_billmsec"),
	} {
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err == nil && parsed > 0 {
			return true
		}
	}
	return false
}

func freeSWITCHEventTime(event freeswitch.Event) (time.Time, error) {
	raw := strings.TrimSpace(event.Header("Event-Date-Timestamp"))
	if raw == "" {
		return time.Now().UTC(), nil
	}

	micros, err := strconv.ParseInt(raw, 10, 64)
	if err != nil {
		return time.Time{}, fmt.Errorf("parse FreeSWITCH event timestamp: %w", err)
	}
	return time.UnixMicro(micros).UTC(), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

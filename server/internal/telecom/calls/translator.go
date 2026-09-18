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
	ErrUnsupportedEvent  = errors.New("unsupported FreeSWITCH call event")
	ErrUncorrelatedEvent = errors.New("uncorrelated FreeSWITCH call event")
)

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

package calls

import (
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

// TranslateEvent converts a raw FreeSWITCH event into a normalized CallEvent.
func TranslateEvent(fsEvent freeswitch.Event, appID, organizationID string) (*CallEvent, error) {
	callID := strings.TrimSpace(fsEvent.Header("Unique-ID"))
	if callID == "" {
		return nil, fmt.Errorf("FreeSWITCH event missing Unique-ID header")
	}

	occurredAt, err := parseTimestamp(fsEvent.Header("Event-Date-Timestamp"))
	if err != nil {
		return nil, fmt.Errorf("parse event timestamp: %w", err)
	}

	from := fsEvent.Header("Caller-Caller-ID-Number")
	to := fsEvent.Header("Caller-Destination-Number")
	direction := determineDirection(fsEvent)

	base := func(eventType CallEventType, status CallStatus) *CallEvent {
		return &CallEvent{
			EventType:      eventType,
			CallID:         callID,
			ApplicationID:  appID,
			OrganizationID: organizationID,
			From:           from,
			To:             to,
			Direction:      direction,
			Status:         status,
			OccurredAt:     occurredAt,
		}
	}

	switch fsEvent.Name {
	case "CHANNEL_CREATE":
		return base(EventCallInitiated, StatusInitiated), nil

	case "CHANNEL_RINGING", "CHANNEL_PROGRESS":
		return base(EventCallRinging, StatusRinging), nil

	case "CHANNEL_ANSWER", "CHANNEL_BRIDGE":
		return base(EventCallAnswered, StatusAnswered), nil

	case "CHANNEL_HANGUP_COMPLETE":
		answered := eventAnswered(fsEvent)
		eventType := EventCallCompleted
		if !answered {
			eventType = EventCallFailed
		}

		event := base(eventType, mapHangupCause(fsEvent.Header("Hangup-Cause")))
		event.DurationSec = calculateDuration(fsEvent, answered)
		return event, nil
	}

	return nil, fmt.Errorf("unhandled FreeSWITCH event: %s", fsEvent.Name)
}

func determineDirection(event freeswitch.Event) CallDirection {
	if direction := strings.TrimSpace(event.Header("Call-Direction")); direction != "" {
		if strings.EqualFold(direction, "inbound") {
			return DirectionInbound
		}
		return DirectionOutbound
	}

	if strings.EqualFold(event.Header("variable_direction"), "inbound") {
		return DirectionInbound
	}
	if strings.EqualFold(event.Header("variable_sip_direction"), "inbound") {
		return DirectionInbound
	}

	return DirectionOutbound
}

func eventAnswered(event freeswitch.Event) bool {
	return strings.EqualFold(strings.TrimSpace(event.Header("Answered")), "true") ||
		strings.EqualFold(strings.TrimSpace(event.Header("variable_answered")), "true")
}

func mapHangupCause(cause string) CallStatus {
	switch strings.ToUpper(strings.TrimSpace(cause)) {
	case "NORMAL_CLEARING", "NORMAL_UNSPECIFIED":
		return StatusCompleted
	case "USER_BUSY", "CALL_REJECTED", "NORMAL_CIRCUIT_CONGESTION":
		return StatusBusy
	case "NO_ANSWER", "ALLOTTED_TIMEOUT", "RECOVERY_ON_TIMER_EXPIRE":
		return StatusNoAnswer
	default:
		return StatusFailed
	}
}

func calculateDuration(event freeswitch.Event, answered bool) int {
	if !answered {
		return 0
	}

	billmsec := parseInt(event.Header("billmsec"))
	if billmsec > 0 {
		return int(billmsec / 1000)
	}

	start, startErr := parseTimestamp(event.Header("start_stamp"))
	end, endErr := parseTimestamp(event.Header("end_stamp"))
	if startErr == nil && endErr == nil && !end.Before(start) {
		return int(end.Sub(start).Seconds())
	}

	return 0
}

func parseTimestamp(value string) (time.Time, error) {
	value = strings.TrimSpace(value)
	if value == "" {
		return time.Time{}, fmt.Errorf("timestamp is empty")
	}

	if micros, err := strconv.ParseInt(value, 10, 64); err == nil {
		return time.UnixMicro(micros).UTC(), nil
	}

	for _, layout := range []string{
		time.RFC3339Nano,
		time.RFC3339,
		"2006-01-02 15:04:05.000000",
		"2006-01-02 15:04:05",
	} {
		if timestamp, err := time.Parse(layout, value); err == nil {
			return timestamp.UTC(), nil
		}
	}

	return time.Time{}, fmt.Errorf("unable to parse timestamp: %q", value)
}

func parseInt(value string) int64 {
	parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
	if err != nil {
		return 0
	}

	return parsed
}

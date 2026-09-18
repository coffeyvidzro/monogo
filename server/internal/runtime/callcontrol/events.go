package callcontrol

import (
	"context"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/google/uuid"
)

var (
	ErrUnsupportedEvent  = errors.New("unsupported FreeSWITCH call event")
	ErrUncorrelatedEvent = errors.New("uncorrelated FreeSWITCH call event")
)

type lifecycleObserver interface {
	ObserveLifecycle(context.Context, calls.LifecycleEvent) error
}

type Consumer struct {
	service lifecycleObserver
}

func NewConsumer(service lifecycleObserver) *Consumer {
	if service == nil {
		panic("callcontrol: lifecycle service is required")
	}
	return &Consumer{service: service}
}

func (c *Consumer) HandleFreeSWITCHEvent(ctx context.Context, event freeswitch.Event) error {
	input, err := TranslateEvent(event)
	if errors.Is(err, ErrUnsupportedEvent) || errors.Is(err, ErrUncorrelatedEvent) {
		return nil
	}
	if err != nil {
		return err
	}
	return c.service.ObserveLifecycle(ctx, input)
}

func TranslateEvent(event freeswitch.Event) (calls.LifecycleEvent, error) {
	eventType, err := lifecycleEventType(event)
	if err != nil {
		return calls.LifecycleEvent{}, err
	}

	rawCallID := strings.TrimSpace(event.Header("variable_leamout_call_id"))
	callID, err := uuid.Parse(rawCallID)
	if err != nil {
		return calls.LifecycleEvent{}, fmt.Errorf("%w: missing valid Leamout call id", ErrUncorrelatedEvent)
	}

	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return calls.LifecycleEvent{}, fmt.Errorf("FreeSWITCH call event is missing Unique-ID")
	}

	occurredAt, err := freeSWITCHEventTime(event)
	if err != nil {
		return calls.LifecycleEvent{}, err
	}

	result := calls.LifecycleEvent{
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

func lifecycleEventType(event freeswitch.Event) (calls.LifecycleEventType, error) {
	switch event.Name {
	case "CHANNEL_CREATE":
		return calls.LifecycleInitiated, nil
	case "CHANNEL_ANSWER":
		return calls.LifecycleAnswered, nil
	case "CHANNEL_HOLD":
		return calls.LifecycleHeld, nil
	case "CHANNEL_UNHOLD":
		return calls.LifecycleResumed, nil
	case "CHANNEL_HANGUP_COMPLETE":
		if eventAnswered(event) {
			return calls.LifecycleCompleted, nil
		}
		if strings.EqualFold(strings.TrimSpace(event.Header("Hangup-Cause")), "ORIGINATOR_CANCEL") {
			return calls.LifecycleCancelled, nil
		}
		return calls.LifecycleFailed, nil
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

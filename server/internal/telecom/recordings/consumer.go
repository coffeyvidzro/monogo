package recordings

import (
	"context"
	"fmt"
	"strconv"
	"strings"
	"time"

	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

const (
	freeSWITCHEventRecordStart = "RECORD_START"
	freeSWITCHEventRecordStop  = "RECORD_STOP"
)

type lifecycleService interface {
	ObserveStarted(context.Context, LifecycleEvent) error
	ObserveStopped(context.Context, LifecycleEvent) error
}

type Consumer struct {
	service lifecycleService
}

func NewConsumer(service *Service) *Consumer {
	if service == nil {
		panic("recordings: service is required")
	}
	return &Consumer{service: service}
}

func (c *Consumer) HandleFreeSWITCHEvent(ctx context.Context, event freeswitch.Event) error {
	if event.Name != freeSWITCHEventRecordStart && event.Name != freeSWITCHEventRecordStop {
		return nil
	}

	input, err := recordingLifecycleEvent(event)
	if err != nil {
		return err
	}

	switch event.Name {
	case freeSWITCHEventRecordStart:
		return c.service.ObserveStarted(ctx, input)
	case freeSWITCHEventRecordStop:
		return c.service.ObserveStopped(ctx, input)
	default:
		return nil
	}
}

func recordingLifecycleEvent(event freeswitch.Event) (LifecycleEvent, error) {
	channelID := strings.TrimSpace(event.Header("Unique-ID"))
	if channelID == "" {
		return LifecycleEvent{}, fmt.Errorf("FreeSWITCH recording event is missing Unique-ID")
	}

	path := firstRecordingPath(
		event.Header("Record-File-Path"),
		event.Header("variable_record_file_path"),
		event.Header("variable_record_path"),
	)
	if path == "" {
		return LifecycleEvent{}, fmt.Errorf("FreeSWITCH recording event is missing Record-File-Path")
	}

	occurredAt := time.Now().UTC()
	if raw := strings.TrimSpace(event.Header("Event-Date-Timestamp")); raw != "" {
		micros, err := strconv.ParseInt(raw, 10, 64)
		if err != nil {
			return LifecycleEvent{}, fmt.Errorf("parse FreeSWITCH recording timestamp: %w", err)
		}
		occurredAt = time.UnixMicro(micros).UTC()
	}

	return LifecycleEvent{
		ChannelID:  channelID,
		Path:       path,
		OccurredAt: occurredAt,
	}, nil
}

func firstRecordingPath(values ...string) string {
	for _, value := range values {
		if value = strings.TrimSpace(value); value != "" {
			return value
		}
	}
	return ""
}

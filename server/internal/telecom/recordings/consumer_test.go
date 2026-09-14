package recordings

import (
	"context"
	"testing"
	"time"

	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

type fakeLifecycleService struct {
	started int
	stopped int
	last    LifecycleEvent
}

func (f *fakeLifecycleService) ObserveStarted(_ context.Context, event LifecycleEvent) error {
	f.started++
	f.last = event
	return nil
}

func (f *fakeLifecycleService) ObserveStopped(_ context.Context, event LifecycleEvent) error {
	f.stopped++
	f.last = event
	return nil
}

func TestConsumerMapsRecordStart(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}
	timestamp := time.Date(2026, time.August, 29, 8, 0, 0, 0, time.UTC)

	err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: freeSWITCHEventRecordStart,
		Headers: map[string]string{
			"Unique-ID":            "channel-1",
			"Record-File-Path":     "/var/lib/freeswitch/recordings/call-1.wav",
			"Event-Date-Timestamp": "1787990400000000",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.started != 1 || service.stopped != 0 {
		t.Fatalf("lifecycle calls = started:%d stopped:%d, want 1/0", service.started, service.stopped)
	}
	if service.last.ChannelID != "channel-1" {
		t.Fatalf("channel = %q, want channel-1", service.last.ChannelID)
	}
	if service.last.Path != "/var/lib/freeswitch/recordings/call-1.wav" {
		t.Fatalf("path = %q", service.last.Path)
	}
	if !service.last.OccurredAt.Equal(timestamp) {
		t.Fatalf("occurred_at = %s, want %s", service.last.OccurredAt, timestamp)
	}
}

func TestConsumerMapsRecordStop(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}

	err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: freeSWITCHEventRecordStop,
		Headers: map[string]string{
			"Unique-ID":        "channel-2",
			"Record-File-Path": "/var/lib/freeswitch/recordings/call-2.wav",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.started != 0 || service.stopped != 1 {
		t.Fatalf("lifecycle calls = started:%d stopped:%d, want 0/1", service.started, service.stopped)
	}
}

func TestConsumerIgnoresUnrelatedEvent(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}

	if err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{Name: "CHANNEL_ANSWER"}); err != nil {
		t.Fatal(err)
	}
	if service.started != 0 || service.stopped != 0 {
		t.Fatalf("unexpected lifecycle calls: %+v", service)
	}
}

func TestRecordingLifecycleEventRequiresIdentity(t *testing.T) {
	tests := []struct {
		name    string
		headers map[string]string
	}{
		{name: "missing channel", headers: map[string]string{"Record-File-Path": "/tmp/test.wav"}},
		{name: "missing path", headers: map[string]string{"Unique-ID": "channel-3"}},
		{name: "invalid timestamp", headers: map[string]string{
			"Unique-ID":            "channel-3",
			"Record-File-Path":     "/tmp/test.wav",
			"Event-Date-Timestamp": "not-a-timestamp",
		}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := recordingLifecycleEvent(freeswitch.Event{
				Name:    freeSWITCHEventRecordStart,
				Headers: tt.headers,
			})
			if err == nil {
				t.Fatal("expected error")
			}
		})
	}
}

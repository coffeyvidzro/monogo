package calls

import (
	"errors"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/freeswitch"
	"github.com/google/uuid"
)

func TestTranslateFreeSWITCHEventKeepsCallAndChannelIdentitySeparate(t *testing.T) {
	callID := uuid.New()
	event, err := TranslateFreeSWITCHEvent(freeswitch.Event{
		Name: "CHANNEL_ANSWER",
		Headers: map[string]string{
			"Unique-ID":                "fs-channel-1",
			"variable_leamout_call_id": callID.String(),
			"Event-Date-Timestamp":     "1787990400000000",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.CallID != callID {
		t.Fatalf("call id = %s, want %s", event.CallID, callID)
	}
	if event.ChannelID != "fs-channel-1" {
		t.Fatalf("channel id = %q, want fs-channel-1", event.ChannelID)
	}
	if event.Type != LifecycleAnswered {
		t.Fatalf("type = %q, want %q", event.Type, LifecycleAnswered)
	}

	wantTime := time.Date(2026, time.August, 29, 8, 0, 0, 0, time.UTC)
	if !event.OccurredAt.Equal(wantTime) {
		t.Fatalf("occurred at = %s, want %s", event.OccurredAt, wantTime)
	}
}

func TestTranslateFreeSWITCHEventRequiresLeamoutCallID(t *testing.T) {
	_, err := TranslateFreeSWITCHEvent(freeswitch.Event{
		Name: "CHANNEL_ANSWER",
		Headers: map[string]string{
			"Unique-ID": "fs-channel-1",
		},
	})
	if !errors.Is(err, ErrUncorrelatedEvent) {
		t.Fatalf("error = %v, want ErrUncorrelatedEvent", err)
	}
}

func TestTranslateFreeSWITCHHangupKeepsCauseAsReason(t *testing.T) {
	callID := uuid.New()
	event, err := TranslateFreeSWITCHEvent(freeswitch.Event{
		Name: "CHANNEL_HANGUP_COMPLETE",
		Headers: map[string]string{
			"Unique-ID":                "fs-channel-1",
			"variable_leamout_call_id": callID.String(),
			"Hangup-Cause":             "USER_BUSY",
		},
	})
	if err != nil {
		t.Fatal(err)
	}

	if event.Type != LifecycleFailed {
		t.Fatalf("type = %q, want %q", event.Type, LifecycleFailed)
	}
	if event.HangupReason == nil || *event.HangupReason != "USER_BUSY" {
		t.Fatalf("hangup reason = %v, want USER_BUSY", event.HangupReason)
	}
}

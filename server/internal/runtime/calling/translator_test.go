package calling

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
		Name:    "CHANNEL_ANSWER",
		Headers: map[string]string{"Unique-ID": "fs-channel-1"},
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

func TestTranslateInboundFreeSWITCHEvent(t *testing.T) {
	organizationID := uuid.New()
	applicationID := uuid.New()
	phoneNumberID := uuid.New()
	bindingID := uuid.New()
	carrierID := uuid.New()

	event, err := TranslateInboundFreeSWITCHEvent(freeswitch.Event{
		Name: "CHANNEL_CREATE",
		Headers: map[string]string{
			"Unique-ID":            "fs-inbound-1",
			"variable_sip_call_id": "sip-call-123",
			"variable_sip_h_X-Leamout-Organization-ID":       organizationID.String(),
			"variable_sip_h_X-Leamout-Carrier-Connection-ID": carrierID.String(),
			"variable_sip_h_X-Leamout-Phone-Number-ID":       phoneNumberID.String(),
			"variable_sip_h_X-Leamout-Voice-Binding-ID":      bindingID.String(),
			"variable_sip_h_X-Leamout-Voice-Application-ID":  applicationID.String(),
			"Caller-Caller-ID-Number":                        "+14155550100",
			"Caller-Destination-Number":                      "+14155550199",
			"Event-Date-Timestamp":                           "1787990400000000",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if event.ChannelID != "fs-inbound-1" || event.SIPCallID != "sip-call-123" {
		t.Fatalf("unexpected inbound identity: %+v", event)
	}
	if event.OrganizationID != organizationID ||
		event.ApplicationID != applicationID ||
		event.PhoneNumberID != phoneNumberID ||
		event.VoiceBindingID != bindingID ||
		event.CarrierConnectionID != carrierID {
		t.Fatalf("unexpected inbound admission identity: %+v", event)
	}
}

func TestTranslateInboundFreeSWITCHEventIgnoresUnrelatedChannel(t *testing.T) {
	_, err := TranslateInboundFreeSWITCHEvent(freeswitch.Event{
		Name:    "CHANNEL_CREATE",
		Headers: map[string]string{"Unique-ID": "unrelated-channel"},
	})
	if !errors.Is(err, ErrNotInboundAdmission) {
		t.Fatalf("error = %v, want ErrNotInboundAdmission", err)
	}
}

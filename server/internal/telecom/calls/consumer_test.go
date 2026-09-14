package calls

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

type fakeLifecycleService struct {
	ensure   int
	answered int
	finished int
	held     int
	resumed  int
	channel  string
	last     InboundCallEvent
}

func (f *fakeLifecycleService) EnsureInbound(_ context.Context, event InboundCallEvent) error {
	f.ensure++
	f.last = event
	return nil
}

func (f *fakeLifecycleService) MarkInboundAnswered(_ context.Context, event InboundCallEvent) error {
	f.answered++
	f.last = event
	return nil
}

func (f *fakeLifecycleService) FinishInbound(_ context.Context, event InboundCallEvent) error {
	f.finished++
	f.last = event
	return nil
}

func (f *fakeLifecycleService) MarkMediaHeld(_ context.Context, channelID string) error {
	f.held++
	f.channel = channelID
	return nil
}

func (f *fakeLifecycleService) MarkMediaResumed(_ context.Context, channelID string) error {
	f.resumed++
	f.channel = channelID
	return nil
}

func TestConsumerIgnoresUntrustedChannel(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}

	err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: "CHANNEL_CREATE",
		Headers: map[string]string{
			"Unique-ID": "channel-1",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.ensure != 0 || service.answered != 0 || service.finished != 0 {
		t.Fatalf("unexpected lifecycle calls: %+v", service)
	}
}

func TestConsumerMapsTrustedInboundHangup(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}
	organizationID := uuid.New()
	applicationID := uuid.New()
	carrierConnectionID := uuid.New()
	phoneNumberID := uuid.New()

	err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: "CHANNEL_HANGUP_COMPLETE",
		Headers: map[string]string{
			"Unique-ID": "channel-2",
			"variable_sip_h_X-Leamout-Organization-ID":       organizationID.String(),
			"variable_sip_h_X-Leamout-Voice-Application-ID":  applicationID.String(),
			"variable_sip_h_X-Leamout-Carrier-Connection-ID": carrierConnectionID.String(),
			"variable_sip_h_X-Leamout-Phone-Number-ID":       phoneNumberID.String(),
			"Caller-Caller-ID-Number":                        "+233201234567",
			"Caller-Destination-Number":                      "+233301234567",
			"Hangup-Cause":                                   "NORMAL_CLEARING",
			"variable_answer_epoch":                          "1787976000",
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if service.finished != 1 {
		t.Fatalf("finished = %d, want 1", service.finished)
	}
	if service.last.OrganizationID != organizationID {
		t.Fatalf("organization = %s, want %s", service.last.OrganizationID, organizationID)
	}
	if service.last.ApplicationID != applicationID {
		t.Fatalf("application = %s, want %s", service.last.ApplicationID, applicationID)
	}
	if service.last.CarrierConnectionID != carrierConnectionID {
		t.Fatalf("carrier connection = %s, want %s", service.last.CarrierConnectionID, carrierConnectionID)
	}
	if service.last.PhoneNumberID != phoneNumberID {
		t.Fatalf("phone number = %s, want %s", service.last.PhoneNumberID, phoneNumberID)
	}
	if service.last.ChannelID != "channel-2" {
		t.Fatalf("channel = %q, want channel-2", service.last.ChannelID)
	}
	if !service.last.WasAnswered {
		t.Fatal("WasAnswered = false, want true")
	}
}

func TestConsumerRejectsTrustedInboundWithoutPhoneNumberID(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}

	err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: "CHANNEL_CREATE",
		Headers: map[string]string{
			"Unique-ID": "channel-missing-number",
			"variable_sip_h_X-Leamout-Organization-ID":       uuid.NewString(),
			"variable_sip_h_X-Leamout-Voice-Application-ID":  uuid.NewString(),
			"variable_sip_h_X-Leamout-Carrier-Connection-ID": uuid.NewString(),
			"Caller-Destination-Number":                      "+233301234567",
		},
	})
	if err == nil {
		t.Fatal("expected missing phone number attribution to fail")
	}
	if service.ensure != 0 {
		t.Fatalf("EnsureInbound calls = %d, want 0", service.ensure)
	}
}

func TestConsumerReconcilesMediaStateWithoutInboundHeaders(t *testing.T) {
	service := &fakeLifecycleService{}
	consumer := &Consumer{service: service}

	if err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: "CHANNEL_HOLD",
		Headers: map[string]string{
			"Unique-ID": "channel-3",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if service.held != 1 || service.channel != "channel-3" {
		t.Fatalf("hold reconciliation = %+v, want channel-3", service)
	}

	if err := consumer.HandleFreeSWITCHEvent(context.Background(), freeswitch.Event{
		Name: "CHANNEL_UNHOLD",
		Headers: map[string]string{
			"Unique-ID": "channel-3",
		},
	}); err != nil {
		t.Fatal(err)
	}
	if service.resumed != 1 || service.channel != "channel-3" {
		t.Fatalf("resume reconciliation = %+v, want channel-3", service)
	}
}

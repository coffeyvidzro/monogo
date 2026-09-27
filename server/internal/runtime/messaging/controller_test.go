package messaging

import (
	"context"
	"testing"

	domain "github.com/coffeyvidzro/monogo/internal/telecom/messaging"
	"github.com/google/uuid"
)

type captureSender struct{ called bool }

func (s *captureSender) Send(context.Context, Request) (Result, error) {
	s.called = true
	return Result{ExternalID: "external-1"}, nil
}

func TestControllerRoutesSMSAndWhatsApp(t *testing.T) {
	for _, channel := range []domain.Channel{domain.ChannelSMS, domain.ChannelMMS, domain.ChannelWhatsApp} {
		sms, whatsapp := &captureSender{}, &captureSender{}
		controller := NewController(sms, whatsapp)
		body := "hello"
		result, err := controller.Send(context.Background(), domain.Route{CarrierConnectionID: uuid.New()}, domain.OutboundMessage{MessageID: uuid.New(), Channel: channel, From: "+12025550100", To: "+12025550101", Body: &body})
		if err != nil {
			t.Fatalf("Send(%q) error = %v", channel, err)
		}
		if result.ProviderMessageID != "external-1" {
			t.Fatalf("Send(%q) result = %#v", channel, result)
		}
		if channel == domain.ChannelWhatsApp && (!whatsapp.called || sms.called) {
			t.Fatalf("WhatsApp selected wrong sender")
		}
		if channel != domain.ChannelWhatsApp && (!sms.called || whatsapp.called) {
			t.Fatalf("SMS selected wrong sender")
		}
	}
}

func TestControllerRejectsUnsupportedChannel(t *testing.T) {
	_, err := NewController(&captureSender{}, &captureSender{}).Send(context.Background(), domain.Route{CarrierConnectionID: uuid.New()}, domain.OutboundMessage{MessageID: uuid.New(), Channel: domain.Channel("email")})
	if err == nil {
		t.Fatal("Send(email) error = nil")
	}
}

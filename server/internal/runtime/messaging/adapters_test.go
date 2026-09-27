package messaging

import (
	"context"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
)

type fakeSMPP struct{ got smpp.SubmitRequest }

func (f *fakeSMPP) Submit(_ context.Context, request smpp.SubmitRequest) (smpp.SubmitResult, error) {
	f.got = request
	return smpp.SubmitResult{MessageID: "smpp-1"}, nil
}

type fakeWhatsApp struct{ got whatsapp.MessageRequest }

func (f *fakeWhatsApp) SendMessage(_ context.Context, request whatsapp.MessageRequest) (whatsapp.MessageResult, error) {
	f.got = request
	return whatsapp.MessageResult{MessageID: "wamid.1"}, nil
}

func TestSMSAdapterTranslatesRuntimeRequest(t *testing.T) {
	client := &fakeSMPP{}
	result, err := NewSMS(client).Send(context.Background(), Request{From: "12025550100", To: "12025550101", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.ExternalID != "smpp-1" || client.got.From != "12025550100" || client.got.To != "12025550101" || client.got.Text != "hello" {
		t.Fatalf("translation result = %#v, request = %#v", result, client.got)
	}
}

func TestWhatsAppAdapterTranslatesRuntimeRequest(t *testing.T) {
	client := &fakeWhatsApp{}
	result, err := NewWhatsApp(client).Send(context.Background(), Request{To: "12025550101", Text: "hello"})
	if err != nil {
		t.Fatalf("Send() error = %v", err)
	}
	if result.ExternalID != "wamid.1" || client.got.To != "12025550101" || client.got.Text != "hello" {
		t.Fatalf("translation result = %#v, request = %#v", result, client.got)
	}
}

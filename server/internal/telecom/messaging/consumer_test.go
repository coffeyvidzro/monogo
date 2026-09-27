package messaging

import (
	"context"
	"errors"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type fakeProvider struct {
	got    OutboundMessage
	result Submission
	err    error
}

func (p *fakeProvider) Send(_ context.Context, _ Route, message OutboundMessage) (Submission, error) {
	p.got = message
	return p.result, p.err
}

type fakeResolver struct{ route Route }

func (r fakeResolver) Resolve(context.Context, sqlc.Message) (Route, error) {
	return r.route, nil
}

type fakeLifecycle struct {
	attributed  sqlc.Message
	submittedID string
	failed      Failure
}

func (f *fakeLifecycle) SetProviderAttribution(context.Context, uuid.UUID, uuid.UUID, uuid.UUID) (sqlc.Message, error) {
	return f.attributed, nil
}
func (f *fakeLifecycle) MarkSubmitted(_ context.Context, _ uuid.UUID, _ uuid.UUID, id string) (sqlc.Message, error) {
	f.submittedID = id
	value := f.attributed
	value.Status = string(StatusSubmitted)
	return value, nil
}
func (f *fakeLifecycle) MarkSent(context.Context, uuid.UUID, uuid.UUID) (sqlc.Message, error) {
	return sqlc.Message{}, nil
}
func (f *fakeLifecycle) MarkDelivered(context.Context, uuid.UUID, uuid.UUID) (sqlc.Message, error) {
	return sqlc.Message{}, nil
}
func (f *fakeLifecycle) MarkUndelivered(context.Context, uuid.UUID, uuid.UUID, Failure) (sqlc.Message, error) {
	return sqlc.Message{}, nil
}
func (f *fakeLifecycle) MarkFailed(_ context.Context, _ uuid.UUID, _ uuid.UUID, failure Failure) (sqlc.Message, error) {
	f.failed = failure
	value := f.attributed
	value.Status = string(StatusFailed)
	return value, nil
}
func (f *fakeLifecycle) CreateInbound(context.Context, InboundRequest) (sqlc.Message, error) {
	return sqlc.Message{}, nil
}
func (f *fakeLifecycle) GetByProviderID(context.Context, uuid.UUID, string) (sqlc.Message, error) {
	return sqlc.Message{}, nil
}

func TestConsumerHandleQueuedUsesProviderNeutralCommand(t *testing.T) {
	organizationID, messageID, connectionID := uuid.New(), uuid.New(), uuid.New()
	body := "hello"
	queued := sqlc.Message{ID: messageID, OrganizationID: organizationID, Channel: "sms", Direction: "outbound", Status: "queued"}
	lifecycle := &fakeLifecycle{attributed: sqlc.Message{ID: messageID, OrganizationID: organizationID, Channel: "sms", Direction: "outbound", Status: "queued", FromAddress: "+12025550100", ToAddress: "+12025550101", Body: &body, Media: []byte("[]")}}
	provider := &fakeProvider{result: Submission{ProviderMessageID: " carrier-42 "}}
	consumer := NewConsumer(lifecycle, fakeResolver{Route{CarrierConnectionID: connectionID}}, NewChannelRouter(ChannelTransports{SMS: provider}))
	got, err := consumer.HandleQueued(context.Background(), queued)
	if err != nil {
		t.Fatalf("HandleQueued() error = %v", err)
	}
	if got.Status != "submitted" || lifecycle.submittedID != "carrier-42" {
		t.Fatalf("submission was not persisted: %#v", got)
	}
	if provider.got.MessageID != messageID || provider.got.Channel != ChannelSMS || provider.got.Body == nil || *provider.got.Body != body {
		t.Fatalf("provider command = %#v", provider.got)
	}
}

func TestConsumerHandleQueuedReturnsProviderErrorForRetry(t *testing.T) {
	messageID, organizationID, connectionID := uuid.New(), uuid.New(), uuid.New()
	lifecycle := &fakeLifecycle{attributed: sqlc.Message{ID: messageID, OrganizationID: organizationID, Channel: "sms", Direction: "outbound", Status: "queued", Media: []byte("[]")}}
	provider := &fakeProvider{err: errors.New("temporary outage")}
	consumer := NewConsumer(lifecycle, fakeResolver{Route{CarrierConnectionID: connectionID}}, NewChannelRouter(ChannelTransports{SMS: provider}))
	_, err := consumer.HandleQueued(context.Background(), sqlc.Message{ID: messageID, OrganizationID: organizationID, Channel: "sms", Direction: "outbound", Status: "queued"})
	if err == nil {
		t.Fatal("HandleQueued() error = nil")
	}
	if lifecycle.submittedID != "" || lifecycle.failed.Code != nil {
		t.Fatal("transient error changed lifecycle")
	}
}

func TestDeliveryAlreadyAccepted(t *testing.T) {
	for _, test := range []struct {
		current  Status
		incoming DeliveryStatus
		want     bool
	}{{StatusSubmitted, DeliverySent, false}, {StatusSent, DeliverySent, true}, {StatusDelivered, DeliverySent, true}, {StatusUndelivered, DeliveryDelivered, true}, {StatusFailed, DeliveryDelivered, true}} {
		if got := deliveryAlreadyAccepted(test.current, test.incoming); got != test.want {
			t.Errorf("deliveryAlreadyAccepted(%q, %q) = %v, want %v", test.current, test.incoming, got, test.want)
		}
	}
}

func TestChannelRouterSelectsCanonicalTransport(t *testing.T) {
	sms, whatsapp, rcs := &fakeProvider{}, &fakeProvider{}, &fakeProvider{}
	router := NewChannelRouter(ChannelTransports{SMS: sms, WhatsApp: whatsapp, RCS: rcs})
	for _, test := range []struct {
		channel Channel
		want    Transport
	}{{ChannelSMS, sms}, {ChannelMMS, sms}, {ChannelWhatsApp, whatsapp}, {ChannelRCS, rcs}} {
		got, err := router.Transport(test.channel)
		if err != nil {
			t.Fatalf("Transport(%q) error = %v", test.channel, err)
		}
		if got != test.want {
			t.Errorf("Transport(%q) selected the wrong adapter", test.channel)
		}
	}
}

func TestChannelRouterRejectsUnconfiguredTransport(t *testing.T) {
	_, err := NewChannelRouter(ChannelTransports{}).Transport(ChannelWhatsApp)
	if err == nil {
		t.Fatal("Transport(whatsapp) error = nil")
	}
}

package messaging

import (
	"strings"
	"testing"

	"github.com/google/uuid"
)

func TestNormalizeCreateRequestSMS(t *testing.T) {
	body := "  hello from Leamout  "
	got, err := normalizeCreateRequest(CreateRequest{
		Channel: Channel(" SMS "),
		From:    " +233200000001 ",
		To:      " +233200000002 ",
		Body:    &body,
	})
	if err != nil {
		t.Fatalf("normalizeCreateRequest() error = %v", err)
	}
	if got.Channel != ChannelSMS {
		t.Fatalf("channel = %q", got.Channel)
	}
	if got.From != "+233200000001" || got.To != "+233200000002" {
		t.Fatalf("addresses = %q -> %q", got.From, got.To)
	}
	if got.Body == nil || *got.Body != "hello from Leamout" {
		t.Fatalf("body = %v", got.Body)
	}
}

func TestNormalizeCreateRequestRequiresPayload(t *testing.T) {
	_, err := normalizeCreateRequest(CreateRequest{
		Channel: ChannelSMS,
		From:    "+233200000001",
		To:      "+233200000002",
	})
	if err == nil {
		t.Fatal("normalizeCreateRequest() error = nil")
	}
}

func TestNormalizeCreateRequestRejectsSMSMedia(t *testing.T) {
	body := "hello"
	_, err := normalizeCreateRequest(CreateRequest{
		Channel: ChannelSMS,
		From:    "+233200000001",
		To:      "+233200000002",
		Body:    &body,
		Media:   []Media{{URL: "https://example.com/image.jpg", ContentType: "image/jpeg"}},
	})
	if err == nil {
		t.Fatal("normalizeCreateRequest() error = nil")
	}
}

func TestNormalizeInboundRequestRequiresProviderIdentity(t *testing.T) {
	body := "hello"
	_, err := normalizeInboundRequest(InboundRequest{
		OrganizationID:        uuid.New(),
		MessagingConnectionID: uuid.New(),
		Channel:               ChannelSMS,
		From:                  "+233200000001",
		To:                    "+233200000002",
		Body:                  &body,
	})
	if err == nil {
		t.Fatal("normalizeInboundRequest() error = nil")
	}
}

func TestNormalizeListRequest(t *testing.T) {
	status := " DELIVERED "
	direction := " OUTBOUND "
	channel := " SMS "
	req := normalizeListRequest(ListRequest{
		Status:    &status,
		Direction: &direction,
		Channel:   &channel,
		Limit:     50,
	})
	if err := validateListRequest(req); err != nil {
		t.Fatalf("validateListRequest() error = %v", err)
	}
	if *req.Status != "delivered" || *req.Direction != "outbound" || *req.Channel != "sms" {
		t.Fatalf("normalized filters = %q %q %q", *req.Status, *req.Direction, *req.Channel)
	}
}

func TestRequestHashChangesWithPayload(t *testing.T) {
	bodyA := "one"
	bodyB := "two"
	a, err := requestHash(CreateRequest{Channel: ChannelSMS, From: "a", To: "b", Body: &bodyA})
	if err != nil {
		t.Fatalf("requestHash(a) error = %v", err)
	}
	b, err := requestHash(CreateRequest{Channel: ChannelSMS, From: "a", To: "b", Body: &bodyB})
	if err != nil {
		t.Fatalf("requestHash(b) error = %v", err)
	}
	if a == b || strings.TrimSpace(a) == "" || len(a) != 64 {
		t.Fatalf("hashes = %q %q", a, b)
	}
}

package tools

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCreateDefaultsWebhook(t *testing.T) {
	req, err := normalizeCreate(CreateRequest{
		Name:        "lookup_customer",
		Description: "Look up a customer",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		EndpointURL: stringPointer("https://example.com/tool"),
	})
	if err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if req.Type != TypeWebhook {
		t.Fatalf("type = %q, want %q", req.Type, TypeWebhook)
	}
}

func TestNormalizeCreateRejectsBuiltinWebhookURL(t *testing.T) {
	_, err := normalizeCreate(CreateRequest{
		Type:        TypeBuiltin,
		Name:        "transfer_call",
		Description: "Transfer the active call",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		EndpointURL: stringPointer("https://example.com/tool"),
	})
	if err == nil {
		t.Fatal("normalizeCreate() error = nil")
	}
}

func stringPointer(value string) *string { return &value }


func TestNormalizeCreateRejectsUnknownBuiltin(t *testing.T) {
	_, err := normalizeCreate(CreateRequest{
		Type:        TypeBuiltin,
		Name:        "unknown_action",
		Description: "Unknown action",
		Parameters:  json.RawMessage(`{"type":"object"}`),
	})
	if err == nil {
		t.Fatal("normalizeCreate() error = nil")
	}
}

func TestNormalizeCreateRejectsHTTPWebhook(t *testing.T) {
	_, err := normalizeCreate(CreateRequest{
		Type:        TypeWebhook,
		Name:        "lookup_customer",
		Description: "Look up a customer",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		EndpointURL: stringPointer("http://example.com/tool"),
	})
	if err == nil {
		t.Fatal("normalizeCreate() error = nil")
	}
}

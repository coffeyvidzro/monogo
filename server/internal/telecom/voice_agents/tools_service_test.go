package voice_agents

import (
	"encoding/json"
	"testing"
)

func TestNormalizeCreateToolDefaultsWebhook(t *testing.T) {
	req, err := normalizeCreateTool(CreateToolRequest{
		Name:        "lookup_customer",
		Description: "Look up a customer",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		EndpointURL: stringPointer("https://example.com/tool"),
	})
	if err != nil {
		t.Fatalf("normalizeCreateTool() error = %v", err)
	}
	if req.Type != ToolTypeWebhook {
		t.Fatalf("type = %q, want %q", req.Type, ToolTypeWebhook)
	}
}

func TestNormalizeCreateToolRejectsBuiltinWebhookURL(t *testing.T) {
	_, err := normalizeCreateTool(CreateToolRequest{
		Type:        ToolTypeBuiltin,
		Name:        "transfer_call",
		Description: "Transfer the active call",
		Parameters:  json.RawMessage(`{"type":"object"}`),
		EndpointURL: stringPointer("https://example.com/tool"),
	})
	if err == nil {
		t.Fatal("normalizeCreateTool() error = nil")
	}
}

func TestNormalizeCreateToolRejectsNonObjectSchema(t *testing.T) {
	_, err := normalizeCreateTool(CreateToolRequest{
		Type:        ToolTypeWebhook,
		Name:        "lookup_customer",
		Description: "Look up a customer",
		Parameters:  json.RawMessage(`[]`),
		EndpointURL: stringPointer("https://example.com/tool"),
	})
	if err == nil {
		t.Fatal("normalizeCreateTool() error = nil")
	}
}

func stringPointer(value string) *string {
	return &value
}

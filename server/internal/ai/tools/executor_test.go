package tools

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/netip"
	"strings"
	"testing"
)

func TestValidateExecutionEndpointRequiresHTTPS(t *testing.T) {
	for _, raw := range []string{
		"http://example.com/tool",
		"https://user:secret@example.com/tool",
		"https://example.com/tool#fragment",
	} {
		if _, err := validateExecutionEndpoint(raw); err == nil {
			t.Fatalf("validateExecutionEndpoint(%q) error = nil", raw)
		}
	}
	if _, err := validateExecutionEndpoint("https://example.com/tool"); err != nil {
		t.Fatalf("validateExecutionEndpoint() error = %v", err)
	}
}

func TestValidateToolAddressRejectsInternalNetworks(t *testing.T) {
	for _, raw := range []string{
		"127.0.0.1",
		"10.0.0.1",
		"172.16.0.1",
		"192.168.1.1",
		"169.254.169.254",
		"100.64.0.1",
		"::1",
		"fc00::1",
		"fe80::1",
	} {
		address := netip.MustParseAddr(raw)
		if err := validateToolAddress(address); err == nil {
			t.Fatalf("validateToolAddress(%s) error = nil", raw)
		}
	}
	if err := validateToolAddress(netip.MustParseAddr("8.8.8.8")); err != nil {
		t.Fatalf("public address rejected: %v", err)
	}
}

func TestValidateToolArgumentsRequiresBoundedObject(t *testing.T) {
	for _, value := range []json.RawMessage{
		nil,
		json.RawMessage(`[]`),
		json.RawMessage(`{"broken"`),
	} {
		if err := validateToolArguments(value); err == nil {
			t.Fatalf("validateToolArguments(%q) error = nil", value)
		}
	}

	tooLarge := json.RawMessage(`{"value":"` + strings.Repeat("a", maxToolArgumentsBytes) + `"}`)
	if err := validateToolArguments(tooLarge); err == nil {
		t.Fatal("oversized arguments error = nil")
	}

	if err := validateToolArguments(json.RawMessage(`{"account_id":"123"}`)); err != nil {
		t.Fatalf("valid arguments rejected: %v", err)
	}
}

func TestReadBoundedToolResponseRejectsOversizedBody(t *testing.T) {
	payload := bytes.Repeat([]byte("x"), maxToolResponseBytes+1)
	if _, err := readBoundedToolResponse(bytes.NewReader(payload)); err == nil {
		t.Fatal("readBoundedToolResponse() error = nil")
	}
}

func TestRejectToolRedirect(t *testing.T) {
	if err := rejectToolRedirect(&http.Request{}, nil); err != http.ErrUseLastResponse {
		t.Fatalf("rejectToolRedirect() = %v", err)
	}
}

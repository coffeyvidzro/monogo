package tools

import (
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/netip"
	"strings"
	"testing"
	"time"
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
	if err := rejectToolRedirect(&http.Request{}, nil); !errors.Is(err, http.ErrUseLastResponse) {
		t.Fatalf("rejectToolRedirect() = %v", err)
	}
}

func TestSameJSONIgnoresObjectKeyOrder(t *testing.T) {
	left := []byte(`{"a":1,"b":"two"}`)
	right := []byte(`{"b":"two","a":1}`)
	if !sameJSON(left, right) {
		t.Fatal("sameJSON() = false")
	}
}

func TestSignToolRequestIsStable(t *testing.T) {
	timestamp := time.Unix(1700000000, 0).UTC()
	first := signToolRequest("secret", []byte(`{"ok":true}`), timestamp)
	second := signToolRequest("secret", []byte(`{"ok":true}`), timestamp)
	if first == "" || first != second || !strings.HasPrefix(first, "v1=") {
		t.Fatalf("signature = %q", first)
	}
}

func TestRequireNoBuiltinArguments(t *testing.T) {
	if err := requireNoBuiltinArguments(json.RawMessage(`{}`)); err != nil {
		t.Fatalf("empty arguments rejected: %v", err)
	}
	if err := requireNoBuiltinArguments(json.RawMessage(`{"unexpected":true}`)); err == nil {
		t.Fatal("non-empty arguments error = nil")
	}
}

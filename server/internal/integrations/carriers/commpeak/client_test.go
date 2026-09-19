package commpeak

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"net/netip"
	"strings"
	"testing"
	"time"
)

func TestListTerminationCDRs(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodGet || r.URL.Path != "/call_records/termination" {
			t.Errorf("unexpected request: %s %s", r.Method, r.URL.Path)
		}
		if r.Header.Get("Authorization") != "test-credential" {
			t.Error("missing authorization")
		}
		if got := r.URL.Query().Get("time_range"); got != "2026-09-01 - 2026-09-02" {
			t.Errorf("time range = %q", got)
		}
		if got := r.URL.Query().Get("page"); got != "2" {
			t.Errorf("page = %q", got)
		}
		if got := r.URL.Query().Get("sip_account_id"); got != "account-1" {
			t.Errorf("account = %q", got)
		}
		_, _ = w.Write([]byte(`{"data":[]}`))
	}))
	defer server.Close()

	client, err := New(Config{Authorization: "test-credential", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	result, err := client.ListTerminationCDRs(context.Background(), TerminationCDRQuery{
		From:         time.Date(2026, 9, 1, 0, 0, 0, 0, time.UTC),
		To:           time.Date(2026, 9, 2, 0, 0, 0, 0, time.UTC),
		Page:         2,
		SIPAccountID: "account-1",
	})
	if err != nil {
		t.Fatal(err)
	}
	if string(result) != `{"data":[]}` {
		t.Fatalf("response = %s", result)
	}
}

func TestCommPeakValidationAndErrors(t *testing.T) {
	if _, err := New(Config{}); err == nil {
		t.Fatal("expected missing credential error")
	}
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_, _ = w.Write([]byte("sensitive upstream data"))
	}))
	defer server.Close()
	client, err := New(Config{Authorization: "secret", BaseURL: server.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTerminationCDRs(context.Background(), TerminationCDRQuery{
		From: time.Now().Add(-time.Hour), To: time.Now(),
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusUnauthorized {
		t.Fatalf("unexpected API error: %v", err)
	}
	if strings.Contains(err.Error(), "sensitive") || strings.Contains(err.Error(), "secret") {
		t.Fatal("API error exposed sensitive data")
	}
}

func TestCommPeakTransportProtection(t *testing.T) {
	if _, err := New(Config{Authorization: "secret", BaseURL: "http://example.com"}); err == nil {
		t.Fatal("expected non-local HTTP rejection")
	}
	var forwarded bool
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		forwarded = r.Header.Get("Authorization") != ""
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	source := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer source.Close()

	client, err := New(Config{Authorization: "secret", BaseURL: source.URL})
	if err != nil {
		t.Fatal(err)
	}
	_, err = client.ListTerminationCDRs(context.Background(), TerminationCDRQuery{
		From: time.Now().Add(-time.Hour), To: time.Now(),
	})
	var apiErr *APIError
	if !errors.As(err, &apiErr) || apiErr.StatusCode != http.StatusFound {
		t.Fatalf("expected blocked redirect, got %v", err)
	}
	if forwarded {
		t.Fatal("redirect exposed the carrier credential")
	}
}

func TestTerminationEndpointValidation(t *testing.T) {
	valid := TerminationConfig{
		Endpoint:    SIPEndpoint{Host: "sip.provider.example", Port: 5061, Transport: "tls"},
		Credentials: &SIPCredentials{Username: "trunk", Password: "secret"},
	}
	target, err := valid.SIPDestination()
	if err != nil || target.Host != valid.Endpoint.Host {
		t.Fatalf("unexpected SIP destination: %+v, %v", target, err)
	}
	invalid := []SIPEndpoint{
		{Host: "https://provider.example", Port: 5061, Transport: "tls"},
		{Host: "provider.example", Transport: "tls"},
		{Host: "provider.example", Port: 5060, Transport: "http"},
		{Host: "bad host", Port: 5060, Transport: "udp"},
	}
	for _, endpoint := range invalid {
		if err := endpoint.Validate(); err == nil {
			t.Fatalf("expected endpoint validation failure for %+v", endpoint)
		}
	}
	valid.Credentials.Password = ""
	if err := valid.Validate(); err == nil {
		t.Fatal("expected invalid credentials rejection")
	}
}

func TestOriginationSourceAllowlist(t *testing.T) {
	config := OriginationConfig{
		Ingress: SIPEndpoint{Host: "sip.leamout.com", Port: 5061, Transport: "tls"},
		Sources: []netip.Prefix{netip.MustParsePrefix("198.51.100.0/24")},
	}
	if err := config.Validate(); err != nil {
		t.Fatal(err)
	}
	if !config.AllowsSource(netip.MustParseAddr("198.51.100.14")) {
		t.Fatal("expected verified signaling source")
	}
	if config.AllowsSource(netip.MustParseAddr("203.0.113.14")) {
		t.Fatal("unexpected non-provider signaling source")
	}
	config.Sources = []netip.Prefix{netip.MustParsePrefix("0.0.0.0/0")}
	if err := config.Validate(); err == nil || config.AllowsSource(netip.MustParseAddr("198.51.100.14")) {
		t.Fatal("catch-all source network must not authorize calls")
	}
}

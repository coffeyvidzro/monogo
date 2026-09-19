package commpeak

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
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

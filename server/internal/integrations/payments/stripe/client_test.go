package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strconv"
	"strings"
	"testing"
	"time"
)

func TestCreateCheckoutSession(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/checkout/sessions" {
			t.Fatalf("path = %q", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("authorization = %q", got)
		}

		payload, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatal(err)
		}
		values, err := url.ParseQuery(string(payload))
		if err != nil {
			t.Fatal(err)
		}
		if got := values.Get("ui_mode"); got != "custom" {
			t.Fatalf("ui_mode = %q", got)
		}
		if got := values.Get("mode"); got != "payment" {
			t.Fatalf("mode = %q", got)
		}
		if got := values.Get("payment_method_types[0]"); got != "card" {
			t.Fatalf("payment_method_types[0] = %q", got)
		}
		if got := values.Get("line_items[0][price_data][unit_amount]"); got != "5000" {
			t.Fatalf("unit_amount = %q", got)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"id": "cs_test_123",
			"client_secret": "cs_test_123_secret",
			"status": "open",
			"payment_status": "unpaid",
			"currency": "usd",
			"amount_total": 5000
		}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("sk_test", "whsec_test")
	cfg.BaseURL = server.URL
	cfg.HTTPClient = server.Client()

	client, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	session, err := client.CreateCheckoutSession(context.Background(), CreateCheckoutSessionRequest{
		Amount:      5000,
		Currency:    "USD",
		Email:       "customer@example.com",
		Reference:   "checkout_123",
		Description: "Wallet top-up",
	})
	if err != nil {
		t.Fatal(err)
	}
	if session.ID != "cs_test_123" {
		t.Fatalf("session ID = %q", session.ID)
	}
	if MapCheckoutStatus(session) != PaymentStatusPending {
		t.Fatalf("status = %q", MapCheckoutStatus(session))
	}
}

func TestParseWebhook(t *testing.T) {
	const secret = "whsec_test"
	client, err := New(DefaultConfig("sk_test", secret))
	if err != nil {
		t.Fatal(err)
	}

	now := time.Unix(1_800_000_000, 0)
	payload := []byte(`{
		"id": "evt_123",
		"type": "checkout.session.completed",
		"data": {
			"object": {
				"id": "cs_123",
				"status": "complete",
				"payment_status": "paid"
			}
		}
	}`)

	timestamp := strconv.FormatInt(now.Unix(), 10)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(timestamp + "." + string(payload)))
	signature := hex.EncodeToString(mac.Sum(nil))
	header := strings.Join([]string{"t=" + timestamp, "v1=" + signature}, ",")

	event, err := client.ParseWebhook(payload, header, now)
	if err != nil {
		t.Fatal(err)
	}
	if event.ID != "evt_123" || event.Type != "checkout.session.completed" {
		t.Fatalf("unexpected event: %#v", event)
	}
	if MapCheckoutStatus(event.Data.Object) != PaymentStatusSucceeded {
		t.Fatalf("status = %q", MapCheckoutStatus(event.Data.Object))
	}
}

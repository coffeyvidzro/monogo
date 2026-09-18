package paystack

import (
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestChargeMobileMoney(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/charge" {
			t.Fatalf("path = %q, want /charge", r.URL.Path)
		}
		if got := r.Header.Get("Authorization"); got != "Bearer sk_test" {
			t.Fatalf("authorization = %q", got)
		}

		var request ChargeRequest
		if err := json.NewDecoder(r.Body).Decode(&request); err != nil {
			t.Fatal(err)
		}
		if request.Amount != 5000 || request.Email != "customer@example.com" {
			t.Fatalf("unexpected charge request: %#v", request)
		}
		if request.MobileMoney == nil || request.MobileMoney.Provider != MobileMoneyMTN {
			t.Fatalf("unexpected mobile money request: %#v", request.MobileMoney)
		}

		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{
			"status": true,
			"message": "Charge attempted",
			"data": {
				"reference": "ref_123",
				"status": "pay_offline",
				"display_text": "Approve the payment"
			}
		}`))
	}))
	defer server.Close()

	cfg := DefaultConfig("sk_test")
	cfg.BaseURL = server.URL
	cfg.HTTPClient = server.Client()

	client, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	response, err := client.ChargeMobileMoney(context.Background(), ChargeRequest{
		Email:  "customer@example.com",
		Amount: 5000,
		MobileMoney: &MobileMoney{
			Phone:    "0551234987",
			Provider: MobileMoneyMTN,
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if response.Data.Reference != "ref_123" {
		t.Fatalf("reference = %q", response.Data.Reference)
	}
	if MapStatus(response.Data.Status) != PaymentStatusProcessing {
		t.Fatalf("status = %q", MapStatus(response.Data.Status))
	}
}

func TestParseWebhook(t *testing.T) {
	client, err := New(DefaultConfig("sk_test"))
	if err != nil {
		t.Fatal(err)
	}

	payload := []byte(`{
		"event": "charge.success",
		"data": {
			"reference": "ref_123",
			"status": "success",
			"amount": 5000,
			"currency": "GHS"
		}
	}`)
	mac := hmac.New(sha512.New, []byte("sk_test"))
	_, _ = mac.Write(payload)
	signature := hex.EncodeToString(mac.Sum(nil))

	event, err := client.ParseWebhook(payload, signature)
	if err != nil {
		t.Fatal(err)
	}
	if event.Event != "charge.success" || event.Data.Reference != "ref_123" {
		t.Fatalf("unexpected event: %#v", event)
	}
}

func TestMapStatus(t *testing.T) {
	tests := map[string]PaymentStatus{
		"success":     PaymentStatusSucceeded,
		"pay_offline": PaymentStatusProcessing,
		"failed":      PaymentStatusFailed,
		"":            PaymentStatusPending,
	}
	for input, want := range tests {
		if got := MapStatus(input); got != want {
			t.Fatalf("MapStatus(%q) = %q, want %q", input, got, want)
		}
	}
}

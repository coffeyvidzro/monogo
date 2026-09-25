package payments

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/integrations/payments/paystack"
	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
)

func TestStripeVerifierChecksReferenceAndFinancialTerms(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		wantPaid bool
		wantErr  bool
	}{
		{"paid", `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":500,"currency":"usd"}`, true, false},
		{"incomplete", `{"id":"cs_test_123","status":"open","payment_status":"unpaid","amount_total":500,"currency":"usd"}`, false, false},
		{"different session", `{"id":"cs_other","status":"complete","payment_status":"paid","amount_total":500,"currency":"usd"}`, false, true},
		{"different amount", `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":600,"currency":"usd"}`, false, true},
		{"different currency", `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":500,"currency":"ghs"}`, false, true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/checkout/sessions/cs_test_123" {
					http.Error(w, "unexpected path", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()
			cfg := stripe.DefaultConfig("sk_test", "whsec_test")
			cfg.BaseURL = server.URL
			client, err := stripe.New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			got, err := (StripeVerifier{Client: client}).Verify(context.Background(), RecoveryCandidate{
				Provider: "stripe", ProviderReference: "cs_test_123", AmountMinor: 500, Currency: "USD",
			})
			if (err != nil) != tc.wantErr || got.Succeeded != tc.wantPaid {
				t.Fatalf("Verify() = %+v, %v; want paid=%v, err=%v", got, err, tc.wantPaid, tc.wantErr)
			}
		})
	}
}

func TestPaystackVerifierChecksReferenceAndFinancialTerms(t *testing.T) {
	tests := []struct {
		name    string
		ref     string
		status  string
		amount  int64
		currency string
		wantPaid bool
		wantErr bool
	}{
		{name: "paid", ref: "ref_123", status: "success", amount: 500, currency: "GHS", wantPaid: true},
		{name: "pending", ref: "ref_123", status: "pending", amount: 500, currency: "GHS"},
		{name: "wrong reference", ref: "different", status: "success", amount: 500, currency: "GHS", wantErr: true},
		{name: "wrong amount", ref: "ref_123", status: "success", amount: 501, currency: "GHS", wantErr: true},
		{name: "wrong currency", ref: "ref_123", status: "success", amount: 500, currency: "USD", wantErr: true},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if !strings.HasSuffix(r.URL.Path, "/ref_123") {
					http.Error(w, "unexpected path", http.StatusNotFound)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprintf(w, `{"status":true,"data":{"reference":%q,"status":%q,"amount":%d,"currency":%q}}`,
					tc.ref, tc.status, tc.amount, tc.currency)
			}))
			defer server.Close()
			cfg := paystack.DefaultConfig("sk_test")
			cfg.BaseURL = server.URL
			client, err := paystack.New(cfg)
			if err != nil {
				t.Fatal(err)
			}
			got, err := (PaystackVerifier{Client: client}).Verify(context.Background(), RecoveryCandidate{
				Provider: "paystack", ProviderReference: "ref_123", AmountMinor: 500, Currency: "GHS",
			})
			if (err != nil) != tc.wantErr || got.Succeeded != tc.wantPaid {
				t.Fatalf("Verify() = %+v, %v; want paid=%v, err=%v", got, err, tc.wantPaid, tc.wantErr)
			}
		})
	}
}

package payments

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/integrations/payments/stripe"
)

func TestStripeProviderVerification(t *testing.T) {
	tests := []struct {
		name    string
		body    string
		wantErr bool
	}{
		{
			name: "paid",
			body: `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":500,"currency":"usd"}`,
		},
		{
			name:    "unpaid",
			body:    `{"id":"cs_test_123","status":"open","payment_status":"unpaid","amount_total":500,"currency":"usd"}`,
			wantErr: true,
		},
		{
			name:    "wrong amount",
			body:    `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":600,"currency":"usd"}`,
			wantErr: true,
		},
		{
			name:    "wrong currency",
			body:    `{"id":"cs_test_123","status":"complete","payment_status":"paid","amount_total":500,"currency":"ghs"}`,
			wantErr: true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Path != "/checkout/sessions/cs_test_123" {
					http.NotFound(w, r)
					return
				}
				w.Header().Set("Content-Type", "application/json")
				_, _ = fmt.Fprint(w, tc.body)
			}))
			defer server.Close()

			cfg := stripe.DefaultConfig("sk_test", "whsec_test")
			cfg.BaseURL = server.URL
			cfg.HTTPClient = server.Client()
			client, err := stripe.New(cfg)
			if err != nil {
				t.Fatal(err)
			}

			providers := NewProviderService(client)
			id := "cs_test_123"
			_, err = providers.Verify(context.Background(), Payment{
				Provider:          "stripe",
				ProviderPaymentID: &id,
				AmountMicros:      5_000_000,
				Currency:          "USD",
			})
			if (err != nil) != tc.wantErr {
				t.Fatalf("Verify() error = %v, wantErr=%v", err, tc.wantErr)
			}
		})
	}
}

func TestMicrosToMinor(t *testing.T) {
	got, err := microsToMinor(5_000_000)
	if err != nil {
		t.Fatal(err)
	}
	if got != 500 {
		t.Fatalf("minor amount = %d, want 500", got)
	}

	if _, err := microsToMinor(1); err == nil {
		t.Fatal("non-representable amount succeeded")
	}
}

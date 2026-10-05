package webhooks

import (
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
)

func TestDeliveryAndTestRejectInternalHTTPS(t *testing.T) {
	endpoint := "https://127.0.0.1:443/internal"
	attempt := NewHTTPSender().Send(t.Context(), sqlc.ClaimWebhookDeliveriesRow{Url: endpoint})
	if attempt.Err == nil {
		t.Fatal("delivery accepted internal destination")
	}
	if _, err := sendTest(t.Context(), sqlc.WebhookEndpoint{Url: endpoint}); err == nil {
		t.Fatal("test delivery accepted internal destination")
	}
}

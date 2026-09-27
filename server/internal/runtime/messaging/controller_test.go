package messaging

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
	"github.com/google/uuid"
)

func TestControllerUsesSelectedWhatsAppConnection(t *testing.T) {
	selected := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"messages":[{"id":"selected"}]}`)
	}))
	defer selected.Close()
	other := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		_, _ = fmt.Fprint(w, `{"messages":[{"id":"other"}]}`)
	}))
	defer other.Close()
	newClient := func(server *httptest.Server) *whatsapp.Client {
		client, err := whatsapp.New(whatsapp.Config{
			BaseURL:       server.URL,
			AccessToken:   "token",
			PhoneNumberID: "phone",
			AppSecret:     "secret",
			Timeout:       time.Second,
		}, server.Client())
		if err != nil {
			t.Fatal(err)
		}
		return client
	}
	selectedID, otherID := uuid.New(), uuid.New()
	controller := NewController()
	controller.RegisterWhatsApp(selectedID, newClient(selected))
	controller.RegisterWhatsApp(otherID, newClient(other))
	result, err := controller.Send(t.Context(), Request{
		MessageID:    uuid.New(),
		ConnectionID: selectedID,
		Channel:      ChannelWhatsApp,
		To:           "15550001",
		Text:         "hello",
	})
	if err != nil {
		t.Fatal(err)
	}
	if result.ExternalID != "selected" {
		t.Fatalf("external id = %q", result.ExternalID)
	}
}

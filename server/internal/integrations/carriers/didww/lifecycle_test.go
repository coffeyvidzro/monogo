package didww

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/telecom/lifecycle"
	"github.com/google/uuid"
)

func TestDIDWWLifecycleProviderRequestsAndVerifiesRelease(t *testing.T) {
	terminated := false
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v3/dids/did-1" {
			t.Fatalf("unexpected path %s", r.URL.Path)
		}
		if r.Method == http.MethodPatch {
			terminated = true
		}
		w.Header().Set("Content-Type", "application/vnd.api+json")
		_, _ = w.Write([]byte(`{"data":{"id":"did-1","type":"dids","attributes":{"number":"12125551234","terminated":` + boolString(terminated) + `}}}`))
	}))
	defer server.Close()
	client, err := New(Config{APIKey: "test", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	provider := NewLifecycleProvider(client)
	reference, err := provider.RequestRelease(context.Background(), "did-1")
	if err != nil || reference != "did-1" {
		t.Fatalf("RequestRelease() = %q, %v", reference, err)
	}
	done, err := provider.ReleaseCompleted(context.Background(), "did-1")
	if err != nil || !done {
		t.Fatalf("ReleaseCompleted() = %v, %v", done, err)
	}
}

func boolString(value bool) string {
	if value {
		return "true"
	}
	return "false"
}

func TestLifecycleProviderUnsupportedCapabilities(t *testing.T) {
	provider := NewLifecycleProvider(nil)
	_, _, _, err := provider.ValidateEmergency(context.Background(), "+15551234567", lifecycle.EmergencyAddressRequest{})
	if !errors.Is(err, lifecycle.ErrProviderCapabilityUnavailable) {
		t.Fatalf("ValidateEmergency() error = %v", err)
	}
	_, err = provider.SubmitPortIn(context.Background(), uuid.New(), lifecycle.PortInRequest{}, nil)
	if !errors.Is(err, lifecycle.ErrProviderCapabilityUnavailable) {
		t.Fatalf("SubmitPortIn() error = %v", err)
	}
	_, _, err = provider.CheckPortability(context.Background(), "+15551234567")
	if !errors.Is(err, lifecycle.ErrProviderCapabilityUnavailable) {
		t.Fatalf("CheckPortability() error = %v", err)
	}
	_, err = provider.PortInStatus(context.Background(), "test-ref")
	if !errors.Is(err, lifecycle.ErrProviderCapabilityUnavailable) {
		t.Fatalf("PortInStatus() error = %v", err)
	}
}

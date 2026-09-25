package lifecycle

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/carriers/didww"
	"github.com/google/uuid"
)

func TestEmergencyAddressNormalizationAndValidation(t *testing.T) {
	request := EmergencyAddressRequest{
		Name: " Jane Doe ", AddressLine1: " 1 Main St ", Locality: " Austin ",
		Region: " tx ", PostalCode: " 78701 ", CountryCode: " us ",
	}
	normalizeEmergency(&request)
	if err := validateEmergency(request); err != nil {
		t.Fatal(err)
	}
	if request.Region != "TX" || request.CountryCode != "US" || request.PostalCode != "78701" {
		t.Fatalf("unexpected normalized address: %+v", request)
	}
	request.PostalCode = "!"
	if err := validateEmergency(request); err == nil {
		t.Fatal("invalid postal code was accepted")
	}
}

func TestLifecycleResponsesHideProviderAndAccountIdentifiers(t *testing.T) {
	providerID := uuid.New()
	providerReference := "private-provider-reference"
	accountNumber := "private-account"
	response := PortInResponse{
		Case:      sqlc.PortInCase{ID: uuid.New(), AccountNumber: accountNumber, ProviderCaseReference: &providerReference},
		Operation: sqlc.NumberLifecycleOperation{ID: uuid.New(), ProviderID: providerID, ProviderReference: &providerReference},
	}
	payload, err := json.Marshal(portInResponse(response))
	if err != nil {
		t.Fatal(err)
	}
	for _, private := range []string{providerID.String(), providerReference, accountNumber} {
		if strings.Contains(string(payload), private) {
			t.Fatalf("private provider data exposed: %s", payload)
		}
	}
}

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
	client, err := didww.New(didww.Config{APIKey: "test", BaseURL: server.URL + "/v3"})
	if err != nil {
		t.Fatal(err)
	}
	provider := NewDIDWWLifecycleProvider(client)
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

package lifecycle

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
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

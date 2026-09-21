package managed

import "testing"

func TestNewAllowsBYOCOnlyDeployment(t *testing.T) {
	providers, err := New(Config{})
	if err != nil {
		t.Fatal(err)
	}
	if providers.DIDWWConfigured() || providers.CommPeakConfigured() {
		t.Fatal("managed providers must remain disabled without platform credentials")
	}
}

func TestNewConfiguresManagedProviderRoles(t *testing.T) {
	providers, err := New(Config{
		DIDWWAPIKey:           "didww-secret",
		DIDWWBaseURL:          "http://localhost:8089/v3",
		CommPeakAuthorization: "commpeak-secret",
		CommPeakBaseURL:       "http://localhost:8090",
	})
	if err != nil {
		t.Fatal(err)
	}
	if !providers.DIDWWConfigured() || !providers.CommPeakConfigured() {
		t.Fatal("both managed carrier roles must be configured")
	}
}

func TestNewRejectsInvalidConfiguredProvider(t *testing.T) {
	_, err := New(Config{DIDWWAPIKey: "key", DIDWWBaseURL: "http://example.com/v3"})
	if err == nil {
		t.Fatal("expected invalid DIDWW transport to fail startup")
	}
}

package providers

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestVerifierUsesProviderAuthenticationAndNormalizesResults(t *testing.T) {
	tests := []struct {
		name              string
		provider          string
		status            int
		wantState         string
		wantFailureCode   string
		wantAuthorization string
		wantAPIKey        string
	}{
		{
			name:              "Deepgram ready",
			provider:          ProviderDeepgram,
			status:            http.StatusOK,
			wantState:         ConnectionReady,
			wantAuthorization: "Token test-secret",
		},
		{
			name:            "Cartesia invalid",
			provider:        ProviderCartesia,
			status:          http.StatusUnauthorized,
			wantState:       ConnectionInvalid,
			wantFailureCode: "authentication_failed",
			wantAPIKey:      "test-secret",
		},
		{
			name:              "Groq rate limited",
			provider:          ProviderGroq,
			status:            http.StatusTooManyRequests,
			wantState:         ConnectionUnavailable,
			wantFailureCode:   "provider_rate_limited",
			wantAuthorization: "Bearer test-secret",
		},
		{
			name:              "OpenAI unavailable",
			provider:          ProviderOpenAI,
			status:            http.StatusServiceUnavailable,
			wantState:         ConnectionUnavailable,
			wantFailureCode:   "provider_unavailable",
			wantAuthorization: "Bearer test-secret",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if got := r.Header.Get("Authorization"); got != test.wantAuthorization {
					t.Errorf("Authorization = %q, want %q", got, test.wantAuthorization)
				}
				if got := r.Header.Get("X-API-Key"); got != test.wantAPIKey {
					t.Errorf("X-API-Key = %q, want %q", got, test.wantAPIKey)
				}
				if test.provider == ProviderCartesia && r.Header.Get("Cartesia-Version") == "" {
					t.Error("Cartesia-Version is empty")
				}
				w.WriteHeader(test.status)
			}))
			defer server.Close()

			verifier := NewVerifier(server.Client())
			verifier.endpoints[test.provider] = server.URL
			result, err := verifier.Verify(context.Background(), test.provider, "test-secret")
			if err != nil {
				t.Fatalf("Verify() error = %v", err)
			}
			if result.State != test.wantState {
				t.Errorf("state = %q, want %q", result.State, test.wantState)
			}
			if result.FailureCode != test.wantFailureCode {
				t.Errorf("failure code = %q, want %q", result.FailureCode, test.wantFailureCode)
			}
		})
	}
}

func TestVerifierPreservesContextCancellation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	verifier := NewVerifier(nil)
	_, err := verifier.Verify(ctx, ProviderOpenAI, "test-secret")
	if !errors.Is(err, context.Canceled) {
		t.Fatalf("Verify() error = %v, want context.Canceled", err)
	}
}

func TestProviderCapabilitiesReturnsDefensiveValues(t *testing.T) {
	capabilities := providerCapabilities(ProviderOpenAI)
	if len(capabilities) == 0 {
		t.Fatal("OpenAI capabilities are empty")
	}
	capabilities[0] = "changed"
	if providerCapabilities(ProviderOpenAI)[0] == "changed" {
		t.Fatal("providerCapabilities() reused mutable storage")
	}
	if got := providerCapabilities("unknown"); len(got) != 0 {
		t.Fatalf("unknown capabilities = %v", got)
	}
}

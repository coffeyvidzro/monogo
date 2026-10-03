package providers

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

const maxVerificationResponse = 4 << 10

type Verifier struct {
	client    *http.Client
	endpoints map[string]string
}

type verificationResult struct {
	State       string
	FailureCode string
}

func NewVerifier(client *http.Client) *Verifier {
	if client == nil {
		client = &http.Client{
			Timeout: 5 * time.Second,
		}
	}
	return &Verifier{
		client: client,
		endpoints: map[string]string{
			ProviderDeepgram: "https://api.deepgram.com/v1/projects",
			ProviderGroq:     "https://api.groq.com/openai/v1/models",
			ProviderCartesia: "https://api.cartesia.ai/models",
			ProviderOpenAI:   "https://api.openai.com/v1/models",
		},
	}
}

func (v *Verifier) Verify(
	ctx context.Context,
	provider string,
	secret string,
) (verificationResult, error) {
	endpoint, ok := v.endpoints[provider]
	if !ok {
		return verificationResult{}, fmt.Errorf("unsupported AI provider %q", provider)
	}
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint, nil)
	if err != nil {
		return verificationResult{}, fmt.Errorf("create provider verification request: %w", err)
	}
	setVerificationHeaders(req, provider, secret)

	resp, err := v.client.Do(req)
	if err != nil {
		if ctx.Err() != nil {
			return verificationResult{}, ctx.Err()
		}
		return verificationResult{
			State:       ConnectionUnavailable,
			FailureCode: "provider_unavailable",
		}, nil
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxVerificationResponse))

	switch {
	case resp.StatusCode >= http.StatusOK && resp.StatusCode < http.StatusMultipleChoices:
		return verificationResult{
			State: ConnectionReady,
		}, nil
	case resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden:
		return verificationResult{
			State:       ConnectionInvalid,
			FailureCode: "authentication_failed",
		}, nil
	case resp.StatusCode == http.StatusTooManyRequests:
		return verificationResult{
			State:       ConnectionUnavailable,
			FailureCode: "provider_rate_limited",
		}, nil
	default:
		return verificationResult{
			State:       ConnectionUnavailable,
			FailureCode: "provider_unavailable",
		}, nil
	}
}

func setVerificationHeaders(req *http.Request, provider string, secret string) {
	secret = strings.TrimSpace(secret)
	switch provider {
	case ProviderDeepgram:
		req.Header.Set("Authorization", "Token "+secret)
	case ProviderCartesia:
		req.Header.Set("X-API-Key", secret)
		req.Header.Set("Cartesia-Version", "2025-04-16")
	default:
		req.Header.Set("Authorization", "Bearer "+secret)
	}
}

func providerCapabilities(provider string) []string {
	switch provider {
	case ProviderDeepgram:
		return []string{
			"streaming",
			"turn_detection",
		}
	case ProviderGroq:
		return []string{
			"streaming",
			"tool_calling",
			"usage",
		}
	case ProviderCartesia:
		return []string{
			"streaming",
		}
	case ProviderOpenAI:
		return []string{
			"streaming",
			"turn_detection",
			"tool_calling",
			"usage",
			"barge_in",
		}
	default:
		return []string{}
	}
}

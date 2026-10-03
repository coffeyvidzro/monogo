package providers

import (
	"encoding/json"
	"testing"
)

func TestValidateProviderConfig(t *testing.T) {
	tests := []struct {
		name     string
		provider string
		config   json.RawMessage
		wantErr  bool
	}{
		{
			name:     "Deepgram",
			provider: ProviderDeepgram,
			config:   json.RawMessage(`{"model":"flux-general-en","language":"en"}`),
		},
		{
			name:     "Groq",
			provider: ProviderGroq,
			config:   json.RawMessage(`{"model":"llama","temperature":0.3}`),
		},
		{
			name:     "Cartesia",
			provider: ProviderCartesia,
			config:   json.RawMessage(`{"model":"sonic-3","voice_id":"voice","language":"en"}`),
		},
		{
			name:     "OpenAI",
			provider: ProviderOpenAI,
			config:   json.RawMessage(`{"model":"gpt-realtime","voice":"alloy"}`),
		},
		{
			name:     "unknown field",
			provider: ProviderOpenAI,
			config:   json.RawMessage(`{"secret":"not-allowed"}`),
			wantErr:  true,
		},
		{
			name:     "wrong type",
			provider: ProviderGroq,
			config:   json.RawMessage(`{"temperature":"hot"}`),
			wantErr:  true,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateProviderConfig(test.provider, test.config)
			if (err != nil) != test.wantErr {
				t.Fatalf("validateProviderConfig() error = %v, wantErr = %v", err, test.wantErr)
			}
		})
	}
}

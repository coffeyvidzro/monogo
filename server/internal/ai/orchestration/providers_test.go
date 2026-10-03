package orchestration

import (
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func TestValidateProviderTopologyReportsMissingBinding(t *testing.T) {
	tests := []struct {
		name      string
		engine    session.Engine
		providers []session.ProviderRuntime
		want      string
	}{
		{
			name:   "realtime",
			engine: session.EngineIntegrated,
			want:   "Voice Agent requires an OpenAI realtime provider binding",
		},
		{
			name:   "stt",
			engine: session.EngineComposable,
			want:   "Voice Agent requires a Deepgram STT provider binding",
		},
		{
			name:   "llm",
			engine: session.EngineComposable,
			providers: []session.ProviderRuntime{
				{
					Role:     "stt",
					Provider: "deepgram",
				},
			},
			want: "Voice Agent requires a Groq LLM provider binding",
		},
		{
			name:   "tts",
			engine: session.EngineComposable,
			providers: []session.ProviderRuntime{
				{
					Role:     "stt",
					Provider: "deepgram",
				},
				{
					Role:     "llm",
					Provider: "groq",
				},
			},
			want: "Voice Agent requires a Cartesia TTS provider binding",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			err := validateProviderTopology(test.providers, test.engine)
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("validateProviderTopology() error = %v, want %q", err, test.want)
			}
		})
	}
}

func TestValidateProviderTopologyAcceptsCompleteBindings(t *testing.T) {
	tests := []struct {
		name      string
		engine    session.Engine
		providers []session.ProviderRuntime
	}{
		{
			name:   "realtime",
			engine: session.EngineIntegrated,
			providers: []session.ProviderRuntime{
				{
					Role:     "realtime",
					Provider: "openai",
				},
			},
		},
		{
			name:   "composable",
			engine: session.EngineComposable,
			providers: []session.ProviderRuntime{
				{
					Role:     "stt",
					Provider: "deepgram",
				},
				{
					Role:     "llm",
					Provider: "groq",
				},
				{
					Role:     "tts",
					Provider: "cartesia",
				},
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if err := validateProviderTopology(test.providers, test.engine); err != nil {
				t.Fatalf("validateProviderTopology() error = %v", err)
			}
		})
	}
}

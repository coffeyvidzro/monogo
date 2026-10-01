package providers_test

import (
	"testing"

	"github.com/coffeyvidzro/monogo/internal/integrations/cartesia"
	"github.com/coffeyvidzro/monogo/internal/integrations/deepgram"
	"github.com/coffeyvidzro/monogo/internal/integrations/groq"
	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

func TestBuiltInProviderRegistryConformance(t *testing.T) {
	descriptors := []providersdk.Descriptor{
		deepgram.Provider{}.Descriptor(),
		groq.Provider{}.Descriptor(),
		cartesia.Provider{}.Descriptor(),
		openai.Provider{}.Descriptor(),
	}

	registry, err := providersdk.NewRegistry(descriptors...)
	if err != nil {
		t.Fatalf("NewRegistry() error = %v", err)
	}

	tests := []struct {
		kind         providersdk.Kind
		id           string
		capabilities []providersdk.Capability
	}{
		{
			kind: providersdk.KindSTT,
			id:   "deepgram",
			capabilities: []providersdk.Capability{
				providersdk.CapabilityStreaming,
				providersdk.CapabilityTurnDetection,
			},
		},
		{
			kind: providersdk.KindLLM,
			id:   "groq",
			capabilities: []providersdk.Capability{
				providersdk.CapabilityStreaming,
				providersdk.CapabilityToolCalling,
				providersdk.CapabilityUsage,
			},
		},
		{
			kind: providersdk.KindTTS,
			id:   "cartesia",
			capabilities: []providersdk.Capability{
				providersdk.CapabilityStreaming,
			},
		},
		{
			kind: providersdk.KindRealtime,
			id:   "openai",
			capabilities: []providersdk.Capability{
				providersdk.CapabilityStreaming,
				providersdk.CapabilityTurnDetection,
				providersdk.CapabilityToolCalling,
				providersdk.CapabilityUsage,
				providersdk.CapabilityBargeIn,
			},
		},
	}

	for _, test := range tests {
		t.Run(string(test.kind)+"/"+test.id, func(t *testing.T) {
			descriptor, ok := registry.Get(test.kind, test.id)
			if !ok {
				t.Fatalf("registry missing %s provider %q", test.kind, test.id)
			}
			if err := descriptor.Validate(); err != nil {
				t.Fatalf("descriptor validation error = %v", err)
			}

			have := make(map[providersdk.Capability]struct{}, len(descriptor.Capabilities))
			for _, capability := range descriptor.Capabilities {
				have[capability] = struct{}{}
			}
			for _, capability := range test.capabilities {
				if _, ok := have[capability]; !ok {
					t.Fatalf("provider missing capability %q", capability)
				}
			}
		})
	}

	if got := len(registry.List(providersdk.KindSTT)); got != 1 {
		t.Fatalf("STT provider count = %d, want 1", got)
	}
	if got := len(registry.List(providersdk.KindLLM)); got != 1 {
		t.Fatalf("LLM provider count = %d, want 1", got)
	}
	if got := len(registry.List(providersdk.KindTTS)); got != 1 {
		t.Fatalf("TTS provider count = %d, want 1", got)
	}
	if got := len(registry.List(providersdk.KindRealtime)); got != 1 {
		t.Fatalf("realtime provider count = %d, want 1", got)
	}
}

func TestRegistryRejectsDuplicateProviderKind(t *testing.T) {
	descriptor := deepgram.Provider{}.Descriptor()
	_, err := providersdk.NewRegistry(descriptor, descriptor)
	if err == nil {
		t.Fatal("NewRegistry() error = nil")
	}
}

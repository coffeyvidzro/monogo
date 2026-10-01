package providers

import (
	"testing"
)

func ConformanceDescriptor(t *testing.T, descriptor Descriptor, kind Kind, id string, capabilities ...Capability) {
	t.Helper()

	if err := descriptor.Validate(); err != nil {
		t.Fatalf("Descriptor().Validate() error = %v", err)
	}
	if descriptor.Kind != kind {
		t.Fatalf("Descriptor().Kind = %q, want %q", descriptor.Kind, kind)
	}
	if descriptor.ID != id {
		t.Fatalf("Descriptor().ID = %q, want %q", descriptor.ID, id)
	}

	have := make(map[Capability]struct{}, len(descriptor.Capabilities))
	for _, capability := range descriptor.Capabilities {
		have[capability] = struct{}{}
	}
	for _, capability := range capabilities {
		if _, ok := have[capability]; !ok {
			t.Fatalf("Descriptor() missing capability %q", capability)
		}
	}
}

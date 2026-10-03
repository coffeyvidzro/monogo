package agents

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestPresetsAreVersionedRuntimeTemplates(t *testing.T) {
	presets := Presets()
	if len(presets) != 4 {
		t.Fatalf("len(Presets()) = %d, want 4", len(presets))
	}
	for _, preset := range presets {
		if preset.Version <= 0 {
			t.Errorf("preset %q version = %d", preset.ID, preset.Version)
		}
		if preset.Engine != EngineComposable && preset.Engine != EngineIntegrated {
			t.Errorf("preset %q engine = %q", preset.ID, preset.Engine)
		}
		if !json.Valid(preset.EngineConfig) {
			t.Errorf("preset %q engine config is invalid", preset.ID)
		}
	}
}

func TestNormalizeCreateAppliesPresetWithoutSharingMutableConfig(t *testing.T) {
	presetID := "fast"
	integrationID := uuid.New()
	request := CreateRequest{
		Name:         "Support",
		Instructions: "Help the caller.",
		Preset:       &presetID,
		Bindings: []ProviderBindingRequest{
			{
				Role:          "llm",
				Provider:      "groq",
				IntegrationID: integrationID,
			},
		},
	}
	normalized, err := normalizeCreate(request)
	if err != nil {
		t.Fatalf("normalizeCreate() error = %v", err)
	}
	if normalized.Engine != EngineComposable || normalized.presetVersion == nil {
		t.Fatalf("normalized request = %+v", normalized)
	}
	if len(normalized.Bindings[0].Config) == 0 {
		t.Fatal("preset provider config was not copied")
	}
	normalized.Bindings[0].Config[0] = 'x'
	again, err := normalizeCreate(request)
	if err != nil {
		t.Fatalf("second normalizeCreate() error = %v", err)
	}
	if again.Bindings[0].Config[0] == 'x' {
		t.Fatal("preset application reused mutable configuration")
	}
}

func TestExpectedProvidersSeparatesEngineTopologies(t *testing.T) {
	composable := expectedProviders(EngineComposable)
	if len(composable) != 3 || composable["stt"] != "deepgram" ||
		composable["llm"] != "groq" || composable["tts"] != "cartesia" {
		t.Fatalf("composable topology = %v", composable)
	}
	realtime := expectedProviders(EngineIntegrated)
	if len(realtime) != 1 || realtime["realtime"] != "openai" {
		t.Fatalf("realtime topology = %v", realtime)
	}
}

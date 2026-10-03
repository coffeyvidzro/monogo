package agents

import "encoding/json"

const presetVersion int32 = 1

type Preset struct {
	ID             string                     `json:"id"`
	Version        int32                      `json:"version"`
	Name           string                     `json:"name"`
	Description    string                     `json:"description"`
	Engine         string                     `json:"engine"`
	EngineConfig   json.RawMessage            `json:"engine_config"`
	ProviderConfig map[string]json.RawMessage `json:"provider_config"`
}

func Presets() []Preset {
	return []Preset{
		{
			ID:           "balanced",
			Version:      presetVersion,
			Name:         "Balanced",
			Description:  "Default quality, latency, and cost balance.",
			Engine:       EngineComposable,
			EngineConfig: json.RawMessage(`{"response_mode":"balanced"}`),
			ProviderConfig: map[string]json.RawMessage{
				"stt": json.RawMessage(`{"model":"flux-general-en"}`),
				"llm": json.RawMessage(`{"model":"llama-3.3-70b-versatile","temperature":0.4}`),
				"tts": json.RawMessage(`{"model":"sonic-3","language":"en"}`),
			},
		},
		{
			ID:           "fast",
			Version:      presetVersion,
			Name:         "Fast",
			Description:  "Lowest practical response latency.",
			Engine:       EngineComposable,
			EngineConfig: json.RawMessage(`{"response_mode":"fast"}`),
			ProviderConfig: map[string]json.RawMessage{
				"stt": json.RawMessage(`{"model":"flux-general-en"}`),
				"llm": json.RawMessage(`{"model":"llama-3.1-8b-instant","temperature":0.2}`),
				"tts": json.RawMessage(`{"model":"sonic-3","language":"en"}`),
			},
		},
		{
			ID:           "high_intelligence",
			Version:      presetVersion,
			Name:         "High Intelligence",
			Description:  "Reasoning quality over minimum latency.",
			Engine:       EngineComposable,
			EngineConfig: json.RawMessage(`{"response_mode":"high_intelligence"}`),
			ProviderConfig: map[string]json.RawMessage{
				"stt": json.RawMessage(`{"model":"flux-general-en"}`),
				"llm": json.RawMessage(`{"model":"llama-3.3-70b-versatile","temperature":0.3}`),
				"tts": json.RawMessage(`{"model":"sonic-3","language":"en"}`),
			},
		},
		{
			ID:           "realtime",
			Version:      presetVersion,
			Name:         "Realtime",
			Description:  "Integrated speech-to-speech interaction.",
			Engine:       EngineIntegrated,
			EngineConfig: json.RawMessage(`{"response_mode":"realtime"}`),
			ProviderConfig: map[string]json.RawMessage{
				"realtime": json.RawMessage(`{"model":"gpt-realtime"}`),
			},
		},
	}
}

func presetByID(id string) (Preset, bool) {
	for _, preset := range Presets() {
		if preset.ID == id {
			return preset, true
		}
	}
	return Preset{}, false
}

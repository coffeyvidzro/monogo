package session

import (
	"encoding/json"
	"testing"

	"github.com/google/uuid"
)

func TestConfigValidate(t *testing.T) {
	valid := Config{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         EngineComposable,
		InputFormat:    AudioFormat{SampleRateHz: 16000, Channels: 1},
		OutputFormat:   AudioFormat{SampleRateHz: 16000, Channels: 1},
		EngineConfig:   json.RawMessage(`{"model":"test"}`),
	}
	if err := valid.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}

	invalid := valid
	invalid.OutputFormat.Channels = 2
	if err := invalid.Validate(); err == nil {
		t.Fatal("Validate() error = nil")
	}
}

func TestAudioFrameValidate(t *testing.T) {
	format := AudioFormat{SampleRateHz: 16000, Channels: 1}
	if err := (AudioFrame{Data: []byte{0, 1}, Format: format}).Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := (AudioFrame{Data: []byte{0}, Format: format}).Validate(); err == nil {
		t.Fatal("Validate() accepted incomplete PCM16 sample")
	}
}

func TestProfileForEngine(t *testing.T) {
	tests := []struct {
		engine Engine
		rate   int
	}{
		{engine: EngineEcho, rate: 16000},
		{engine: EngineComposable, rate: 16000},
		{engine: EngineIntegrated, rate: 24000},
	}
	for _, tt := range tests {
		t.Run(string(tt.engine), func(t *testing.T) {
			profile, err := ProfileForEngine(tt.engine)
			if err != nil {
				t.Fatalf("ProfileForEngine() error = %v", err)
			}
			if profile.InputFormat.SampleRateHz != tt.rate || profile.OutputFormat.SampleRateHz != tt.rate {
				t.Fatalf("profile = %+v", profile)
			}
		})
	}
}

func TestConfigRejectsInvalidEngineConfigAndProfile(t *testing.T) {
	profile, err := ProfileForEngine(EngineIntegrated)
	if err != nil {
		t.Fatal(err)
	}
	cfg := Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: EngineIntegrated, InputFormat: profile.InputFormat, OutputFormat: profile.OutputFormat,
		EngineConfig: json.RawMessage(`[]`),
	}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted non-object engine config")
	}
	cfg.EngineConfig = json.RawMessage(`{}`)
	cfg.InputFormat = AudioFormat{SampleRateHz: 16000, Channels: 1}
	if err := cfg.Validate(); err == nil {
		t.Fatal("Validate() accepted mismatched engine profile")
	}
}

func TestConfigEqualIncludesEngineConfig(t *testing.T) {
	profile, err := ProfileForEngine(EngineComposable)
	if err != nil {
		t.Fatal(err)
	}
	left := Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: EngineComposable, InputFormat: profile.InputFormat, OutputFormat: profile.OutputFormat,
		EngineConfig: json.RawMessage(`{"model":"a"}`),
	}
	right := left
	right.EngineConfig = json.RawMessage(`{"model":"b"}`)
	if left.Equal(right) {
		t.Fatal("Equal() ignored engine config")
	}
	right.EngineConfig = append(json.RawMessage(nil), left.EngineConfig...)
	if !left.Equal(right) {
		t.Fatal("Equal() rejected identical config")
	}
}

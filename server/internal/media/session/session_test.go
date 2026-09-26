package session

import (
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

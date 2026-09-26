package echo

import (
	"context"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestEngineCopiesAudio(t *testing.T) {
	format := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	stream, err := (Engine{}).Start(context.Background(), session.Config{
		ID: uuid.New(), OrganizationID: uuid.New(), CallID: uuid.New(), ChannelID: uuid.New(),
		Engine: session.EngineEcho, InputFormat: format, OutputFormat: format,
	})
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	source := []byte{1, 0}
	if err := stream.SendAudio(context.Background(), session.AudioFrame{Data: source, Format: format}); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}
	source[0] = 9
	got := <-stream.Audio()
	if got.Data[0] != 1 {
		t.Fatalf("echo frame was not copied: %v", got.Data)
	}
	if err := stream.Close(context.Background()); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
}

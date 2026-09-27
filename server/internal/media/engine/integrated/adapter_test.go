package integrated

import (
	"context"
	"encoding/base64"
	"encoding/binary"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

func TestEngineAdapts16kTransportToOpenAI24k(t *testing.T) {
	updateReceived := make(chan openai.ClientEvent, 1)
	inputReceived := make(chan []byte, 1)
	providerAudio := pcm16Samples(2400, 7000)
	server := httptest.NewTLSServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		ws, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer func() { _ = ws.CloseNow() }()

		_, payload, err := ws.Read(r.Context())
		if err != nil {
			return
		}
		var update openai.ClientEvent
		if err := json.Unmarshal(payload, &update); err != nil {
			return
		}
		updateReceived <- update

		_, payload, err = ws.Read(r.Context())
		if err != nil {
			return
		}
		var appendEvent openai.ClientEvent
		if err := json.Unmarshal(payload, &appendEvent); err != nil {
			return
		}
		decoded, err := base64.StdEncoding.DecodeString(appendEvent.Audio)
		if err != nil {
			return
		}
		inputReceived <- decoded

		response, _ := json.Marshal(openai.ServerEvent{
			Type:    "response.audio.delta",
			EventID: "evt-audio",
			Delta:   base64.StdEncoding.EncodeToString(providerAudio),
		})
		if err := ws.Write(r.Context(), websocket.MessageText, response); err != nil {
			return
		}
		<-r.Context().Done()
	}))
	defer server.Close()

	transportFormat := session.AudioFormat{SampleRateHz: 16000, Channels: 1}
	cfg := session.Config{
		ID:             uuid.New(),
		OrganizationID: uuid.New(),
		CallID:         uuid.New(),
		ChannelID:      uuid.New(),
		Engine:         session.EngineIntegrated,
		InputFormat:    transportFormat,
		OutputFormat:   transportFormat,
	}
	engine := Engine{
		Client: openai.NewClient(server.Client()),
		Config: openai.Config{
			APIKey:   "secret",
			Endpoint: "wss" + strings.TrimPrefix(server.URL, "https"),
		},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()
	stream, err := engine.Start(ctx, cfg)
	if err != nil {
		t.Fatalf("Start() error = %v", err)
	}
	defer func() { _ = stream.Close(context.Background()) }()

	select {
	case update := <-updateReceived:
		if update.Session == nil {
			t.Fatal("session update = nil")
		}
		if got := update.Session.Audio.Input.Format.Rate; got != 24000 {
			t.Fatalf("provider input rate = %d, want 24000", got)
		}
		if got := update.Session.Audio.Output.Format.Rate; got != 24000 {
			t.Fatalf("provider output rate = %d, want 24000", got)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for provider session update")
	}

	input := pcm16Samples(320, 3000)
	if err := stream.SendAudio(ctx, session.AudioFrame{
		Data:       input,
		Format:     transportFormat,
		CapturedAt: time.Now().UTC(),
	}); err != nil {
		t.Fatalf("SendAudio() error = %v", err)
	}

	select {
	case providerInput := <-inputReceived:
		if len(providerInput) <= len(input) {
			t.Fatalf("provider input bytes = %d, want more than 16 kHz input %d", len(providerInput), len(input))
		}
		for offset := 0; offset < len(providerInput); offset += 2 {
			if got := int16(binary.LittleEndian.Uint16(providerInput[offset : offset+2])); got != 3000 {
				t.Fatalf("provider input sample = %d, want 3000", got)
			}
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for provider input audio")
	}

	select {
	case frame := <-stream.Audio():
		if frame.Format != transportFormat {
			t.Fatalf("transport output format = %+v, want %+v", frame.Format, transportFormat)
		}
		if got, want := len(frame.Data), 1600*2; got != want {
			t.Fatalf("transport output bytes = %d, want %d", got, want)
		}
	case <-ctx.Done():
		t.Fatal("timed out waiting for transport output audio")
	}
}

func pcm16Samples(count int, value int16) []byte {
	data := make([]byte, count*2)
	for offset := 0; offset < len(data); offset += 2 {
		binary.LittleEndian.PutUint16(data[offset:offset+2], uint16(value))
	}
	return data
}

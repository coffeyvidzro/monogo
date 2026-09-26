package openai

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/google/uuid"
)

type Client struct {
	httpClient *http.Client
}

func NewClient(httpClient *http.Client) *Client {
	if httpClient == nil {
		httpClient = http.DefaultClient
	}
	return &Client{httpClient: httpClient}
}

func (c *Client) Start(ctx context.Context, cfg Config, sessionConfig session.Config) (session.Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("OpenAI Realtime context is required")
	}
	if strings.TrimSpace(cfg.APIKey) == "" {
		return nil, fmt.Errorf("OpenAI API key is required")
	}
	if err := sessionConfig.Validate(); err != nil {
		return nil, err
	}
	inputFormat := sessionConfig.InputFormat
	outputFormat := sessionConfig.OutputFormat
	if inputFormat.SampleRateHz != 24000 ||
		outputFormat.SampleRateHz != 24000 ||
		inputFormat.Channels != 1 ||
		outputFormat.Channels != 1 {
		return nil, fmt.Errorf("OpenAI Realtime requires mono 24 kHz PCM input and output")
	}
	endpoint := strings.TrimSpace(cfg.Endpoint)
	if endpoint == "" {
		endpoint = DefaultEndpoint
	}
	parsed, err := url.Parse(endpoint)
	if err != nil || parsed.Scheme != "wss" || parsed.Host == "" {
		return nil, fmt.Errorf("OpenAI Realtime endpoint must be an absolute wss URL")
	}
	model := strings.TrimSpace(cfg.Model)
	if model == "" {
		model = DefaultModel
	}
	query := parsed.Query()
	query.Set("model", model)
	parsed.RawQuery = query.Encode()
	header := http.Header{
		"Authorization": []string{"Bearer " + strings.TrimSpace(cfg.APIKey)},
	}
	if safety := strings.TrimSpace(cfg.SafetyIdentifier); safety != "" {
		header.Set("OpenAI-Safety-Identifier", safety)
	}
	connection, response, err := websocket.Dial(ctx, parsed.String(), &websocket.DialOptions{
		HTTPClient:      c.httpClient,
		HTTPHeader:      header,
		CompressionMode: websocket.CompressionDisabled,
	})
	if response != nil && response.Body != nil {
		defer func() {
			_ = response.Body.Close()
		}()
	}
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("connect OpenAI Realtime: HTTP %d: %w", response.StatusCode, err)
		}
		return nil, fmt.Errorf("connect OpenAI Realtime: %w", err)
	}
	streamCtx, cancel := context.WithCancel(ctx)
	result := &realtimeStream{
		ctx:        streamCtx,
		cancel:     cancel,
		connection: connection,
		format:     sessionConfig.OutputFormat,
		events:     make(chan session.Event, 64),
		audio:      make(chan session.AudioFrame, 32),
	}
	update := ClientEvent{
		Type:    "session.update",
		EventID: uuid.NewString(),
		Session: &SessionUpdate{
			Type:             "realtime",
			Instructions:     sessionConfig.Instructions,
			OutputModalities: []string{"audio"},
			Audio: AudioConfig{
				Input: AudioInput{
					Format: AudioFormat{Type: "audio/pcm", Rate: 24000},
				},
				Output: AudioOutput{
					Format: AudioFormat{Type: "audio/pcm", Rate: 24000},
					Voice:  strings.TrimSpace(cfg.Voice),
				},
			},
			Tools: cfg.Tools,
		},
	}
	if err := result.writeJSON(ctx, update); err != nil {
		_ = result.Close(context.Background())
		return nil, fmt.Errorf("configure OpenAI Realtime session: %w", err)
	}
	go result.readLoop()
	return result, nil
}

type realtimeStream struct {
	ctx        context.Context
	cancel     context.CancelFunc
	connection *websocket.Conn
	format     session.AudioFormat
	events     chan session.Event
	audio      chan session.AudioFrame
	writeMu    sync.Mutex
	closeOnce  sync.Once
}

func (s *realtimeStream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	if frame.Format != s.format {
		return fmt.Errorf("OpenAI Realtime audio format changed during stream")
	}
	return s.writeJSON(ctx, ClientEvent{
		Type:    "input_audio_buffer.append",
		EventID: uuid.NewString(),
		Audio:   base64.StdEncoding.EncodeToString(frame.Data),
	})
}

func (s *realtimeStream) Interrupt(ctx context.Context) error {
	return s.writeJSON(ctx, ClientEvent{
		Type:    "response.cancel",
		EventID: uuid.NewString(),
	})
}

func (s *realtimeStream) Audio() <-chan session.AudioFrame {
	return s.audio
}

func (s *realtimeStream) Events() <-chan session.Event {
	return s.events
}

func (s *realtimeStream) Close(context.Context) error {
	var err error
	s.closeOnce.Do(func() {
		s.cancel()
		err = s.connection.CloseNow()
	})
	return err
}

func (s *realtimeStream) writeJSON(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.connection.Write(ctx, websocket.MessageText, payload)
}

func (s *realtimeStream) readLoop() {
	defer close(s.events)
	defer close(s.audio)
	defer func() {
		_ = s.Close(context.Background())
	}()
	for {
		kind, payload, err := s.connection.Read(s.ctx)
		if err != nil {
			if s.ctx.Err() == nil && websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				s.emitEvent(session.Event{
					Type:       session.EventError,
					Text:       fmt.Sprintf("read OpenAI Realtime: %v", err),
					OccurredAt: time.Now().UTC(),
				})
			}
			return
		}
		if kind != websocket.MessageText {
			s.emitEvent(session.Event{
				Type:       session.EventError,
				Text:       "OpenAI Realtime returned a non-text event",
				OccurredAt: time.Now().UTC(),
			})
			return
		}
		var event ServerEvent
		if err := json.Unmarshal(payload, &event); err != nil {
			s.emitEvent(session.Event{
				Type:       session.EventError,
				Text:       fmt.Sprintf("decode OpenAI Realtime event: %v", err),
				OccurredAt: time.Now().UTC(),
			})
			return
		}
		s.handle(event)
	}
}
func (s *realtimeStream) handle(event ServerEvent) {
	now := time.Now().UTC()
	providerID := event.EventID
	switch event.Type {
	case "input_audio_buffer.speech_started":
		s.emitEvent(session.Event{
			Type:       session.EventSpeechStarted,
			ProviderID: providerID,
			OccurredAt: now,
		})
	case "input_audio_buffer.speech_stopped":
		s.emitEvent(session.Event{
			Type:       session.EventSpeechStopped,
			ProviderID: providerID,
			OccurredAt: now,
		})
	case "conversation.item.input_audio_transcription.delta":
		s.emitEvent(session.Event{
			Type:       session.EventTranscriptDelta,
			Text:       event.Delta,
			ProviderID: providerID,
			OccurredAt: now,
		})
	case "conversation.item.input_audio_transcription.completed":
		s.emitEvent(session.Event{
			Type:       session.EventTranscriptFinal,
			Text:       event.Transcript,
			ProviderID: providerID,
			OccurredAt: now,
		})
	case "response.created":
		s.emitEvent(session.Event{Type: session.EventResponseStarted, ProviderID: providerID, OccurredAt: now})
	case "response.done":
		s.emitEvent(session.Event{
			Type:            session.EventResponseStopped,
			ProviderID:      providerID,
			ProviderPayload: marshalRaw(event.Response),
			OccurredAt:      now,
		})
	case "response.audio.delta", "response.output_audio.delta":
		audio, err := base64.StdEncoding.DecodeString(event.Delta)
		if err != nil {
			s.emitEvent(session.Event{Type: session.EventError, Text: err.Error(), OccurredAt: now})
			return
		}
		frame := session.AudioFrame{
			Data:       audio,
			Format:     s.format,
			CapturedAt: now,
		}
		select {
		case s.audio <- frame:
		case <-s.ctx.Done():
		}
	case "response.function_call_arguments.delta", "response.function_call_arguments.done":
		s.emitEvent(session.Event{
			Type:       session.EventToolCall,
			Text:       event.Arguments + event.Delta,
			ProviderID: event.CallID,
			OccurredAt: now,
		})
	case "error":
		if event.Error != nil {
			s.emitEvent(session.Event{
				Type:       session.EventError,
				Text:       event.Error.Code + ": " + event.Error.Message,
				ProviderID: providerID,
				OccurredAt: now,
			})
		}
	}
}
func (s *realtimeStream) emitEvent(event session.Event) {
	select {
	case s.events <- event:
	case <-s.ctx.Done():
	}
}
func marshalRaw(value any) []byte {
	payload, _ := json.Marshal(value)
	return payload
}

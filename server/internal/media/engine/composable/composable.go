// Package composable orchestrates provider-neutral speech-to-text, language
// model, and text-to-speech adapters as one realtime media session.
package composable

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/cartesia"
	"github.com/coffeyvidzro/monogo/internal/integrations/deepgram"
	"github.com/coffeyvidzro/monogo/internal/integrations/groq"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Engine struct {
	Transcriber deepgram.Transcriber
	Generator   groq.Generator
	Synthesizer cartesia.StreamingSynthesizer
	Deepgram    deepgram.Config
	Groq        groq.Config
	Cartesia    cartesia.Config
}

func (e Engine) Start(ctx context.Context, cfg session.Config) (session.Stream, error) {
	if ctx == nil {
		return nil, fmt.Errorf("composable engine context is required")
	}
	if cfg.Engine != session.EngineComposable {
		return nil, fmt.Errorf("composable engine cannot start session engine %q", cfg.Engine)
	}
	transcriber := e.Transcriber
	if transcriber == nil {
		transcriber = deepgram.NewClient(nil)
	}
	generator := e.Generator
	if generator == nil {
		generator = groq.NewClient(nil)
	}
	synthesizer := e.Synthesizer
	if synthesizer == nil {
		synthesizer = cartesia.NewClient(nil)
	}
	deepgramConfig := e.Deepgram
	if language := strings.TrimSpace(cfg.Language); language != "" {
		deepgramConfig.LanguageHints = append(append([]string(nil), deepgramConfig.LanguageHints...), language)
	}
	deepgramStream, err := transcriber.Start(ctx, deepgramConfig, cfg.InputFormat)
	if err != nil {
		return nil, fmt.Errorf("start transcription: %w", err)
	}
	streamCtx, cancel := context.WithCancel(ctx)
	cartesiaConfig := e.Cartesia
	if voice := strings.TrimSpace(cfg.Voice); voice != "" {
		cartesiaConfig.VoiceID = voice
	}
	if language := strings.TrimSpace(cfg.Language); language != "" {
		cartesiaConfig.Language = language
	}
	s := &stream{
		ctx: streamCtx, cancel: cancel, transcriber: deepgramStream,
		generator: generator, synthesizer: synthesizer, groq: e.Groq,
		cartesia: cartesiaConfig, config: cfg,
		audio: make(chan session.AudioFrame), events: make(chan session.Event, 32), done: make(chan struct{}),
	}
	if instructions := strings.TrimSpace(cfg.Instructions); instructions != "" {
		s.messages = append(s.messages, groq.Message{Role: "system", Content: instructions})
	}
	go s.run()
	return s, nil
}

type stream struct {
	ctx         context.Context
	cancel      context.CancelFunc
	transcriber deepgram.Stream
	generator   groq.Generator
	synthesizer cartesia.StreamingSynthesizer
	groq        groq.Config
	cartesia    cartesia.Config
	config      session.Config
	audio       chan session.AudioFrame
	events      chan session.Event

	mu             sync.Mutex
	messages       []groq.Message
	generation     uint64
	responseCancel context.CancelFunc
	responseActive bool
	responses      sync.WaitGroup
	closeOnce      sync.Once
	done           chan struct{}
}

func (s *stream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	return s.transcriber.SendAudio(ctx, frame)
}

func (s *stream) Interrupt(context.Context) error {
	s.cancelResponse()
	return nil
}

func (s *stream) Audio() <-chan session.AudioFrame { return s.audio }
func (s *stream) Events() <-chan session.Event     { return s.events }

func (s *stream) Close(ctx context.Context) error {
	s.closeOnce.Do(func() { s.cancel() })
	if err := s.transcriber.Close(ctx); err != nil && ctx.Err() == nil {
		return err
	}
	select {
	case <-s.done:
		return nil
	case <-ctx.Done():
		return ctx.Err()
	}
}

func (s *stream) run() {
	defer close(s.done)
	defer close(s.audio)
	defer close(s.events)
	for {
		select {
		case event, ok := <-s.transcriber.Events():
			if !ok {
				s.cancelResponse()
				s.responses.Wait()
				return
			}
			if event.Err != nil {
				s.emit(session.Event{Type: session.EventError, Text: event.Err.Error(), OccurredAt: time.Now().UTC()})
				continue
			}
			s.handleTurn(event)
		case <-s.ctx.Done():
			s.cancelResponse()
			s.responses.Wait()
			return
		}
	}
}

func (s *stream) handleTurn(event deepgram.Event) {
	switch event.TurnEvent {
	case "StartOfTurn":
		s.emit(session.Event{Type: session.EventSpeechStarted, ProviderID: event.RequestID, OccurredAt: time.Now().UTC()})
		// Publish speech first so the transport clears already-buffered playback;
		// generation cancellation below fences all subsequent provider audio.
		s.cancelResponse()
	case "EndOfTurn":
		text := strings.TrimSpace(event.Transcript.Text)
		s.emit(session.Event{Type: session.EventSpeechStopped, ProviderID: event.RequestID, OccurredAt: time.Now().UTC()})
		if text == "" {
			return
		}
		s.emit(session.Event{Type: session.EventTranscriptFinal, Text: text, ProviderID: event.RequestID, OccurredAt: time.Now().UTC()})
		s.startResponse(text)
	}
}

func (s *stream) startResponse(text string) {
	s.mu.Lock()
	if s.responseCancel != nil {
		s.responseCancel()
	}
	s.generation++
	generation := s.generation
	responseCtx, cancel := context.WithCancel(s.ctx)
	s.responseCancel = cancel
	s.responseActive = true
	s.messages = append(s.messages, groq.Message{Role: "user", Content: text})
	messages := append([]groq.Message(nil), s.messages...)
	s.responses.Add(1)
	s.mu.Unlock()
	s.emit(session.Event{Type: session.EventResponseStarted, OccurredAt: time.Now().UTC()})
	go s.generate(responseCtx, generation, messages)
}

func (s *stream) generate(ctx context.Context, generation uint64, messages []groq.Message) {
	defer s.responses.Done()
	completion, err := s.generator.Generate(ctx, s.groq, messages)
	if err != nil {
		s.failResponse(ctx, generation, err)
		return
	}
	defer func() { _ = completion.Close() }()
	voice, err := s.synthesizer.StartSynthesis(ctx, s.cartesia, s.config.OutputFormat)
	if err != nil {
		s.failResponse(ctx, generation, err)
		return
	}
	defer func() { _ = voice.Close() }()
	var text strings.Builder
	var pending strings.Builder
	var heldChunk string
	completionEvents := completion.Events()
	for {
		select {
		case event, ok := <-completionEvents:
			if !ok || event.Done {
				finalChunk := strings.TrimSpace(pending.String())
				switch {
				case heldChunk != "" && finalChunk != "":
					if err := voice.SendText(ctx, heldChunk, true); err != nil {
						s.failResponse(ctx, generation, err)
						return
					}
					if err := voice.SendText(ctx, finalChunk, false); err != nil {
						s.failResponse(ctx, generation, err)
						return
					}
				case heldChunk != "":
					if err := voice.SendText(ctx, heldChunk, false); err != nil {
						s.failResponse(ctx, generation, err)
						return
					}
				case finalChunk != "":
					if err := voice.SendText(ctx, finalChunk, false); err != nil {
						s.failResponse(ctx, generation, err)
						return
					}
				default:
					s.stopResponse(generation, "")
					return
				}
				pending.Reset()
				heldChunk = ""
				completionEvents = nil
				continue
			}
			if event.Err != nil {
				s.failResponse(ctx, generation, event.Err)
				return
			}
			if event.TextDelta != "" {
				text.WriteString(event.TextDelta)
				s.emitCurrent(ctx, generation, session.Event{Type: session.EventResponseDelta, Text: event.TextDelta, ProviderID: event.CompletionID, OccurredAt: time.Now().UTC()})
				pending.WriteString(event.TextDelta)
				if shouldFlushSpeechChunk(pending.String()) {
					chunk := strings.TrimSpace(pending.String())
					pending.Reset()
					if heldChunk != "" {
						if err := voice.SendText(ctx, heldChunk, true); err != nil {
							s.failResponse(ctx, generation, err)
							return
						}
					}
					heldChunk = chunk
				}
			}
			if event.ToolName != "" {
				s.emitCurrent(ctx, generation, session.Event{Type: session.EventToolCall, Text: event.ToolName, ProviderID: event.ToolCallID, ProviderPayload: event.ToolArguments, OccurredAt: time.Now().UTC()})
			}
		case <-ctx.Done():
			return
		case event, ok := <-voice.Events():
			if !ok || event.Done {
				s.stopResponse(generation, strings.TrimSpace(text.String()))
				return
			}
			if event.Err != nil {
				s.failResponse(ctx, generation, event.Err)
				return
			}
			if len(event.Audio.Data) != 0 && s.isCurrent(generation) {
				select {
				case s.audio <- event.Audio:
				case <-ctx.Done():
					return
				case <-s.ctx.Done():
					return
				}
			}
		}
	}
}

func (s *stream) cancelResponse() {
	s.mu.Lock()
	s.generation++
	cancel, active := s.responseCancel, s.responseActive
	s.responseCancel = nil
	s.responseActive = false
	s.mu.Unlock()
	if cancel != nil {
		cancel()
	}
	if active {
		s.emit(session.Event{Type: session.EventResponseStopped, OccurredAt: time.Now().UTC()})
	}
}

func (s *stream) stopResponse(generation uint64, assistantText string) {
	s.mu.Lock()
	if generation != s.generation || !s.responseActive {
		s.mu.Unlock()
		return
	}
	s.responseActive = false
	s.responseCancel = nil
	if assistantText != "" {
		s.messages = append(s.messages, groq.Message{Role: "assistant", Content: assistantText})
	}
	s.mu.Unlock()
	s.emit(session.Event{Type: session.EventResponseStopped, OccurredAt: time.Now().UTC()})
}

func (s *stream) failResponse(ctx context.Context, generation uint64, err error) {
	if ctx.Err() != nil || !s.isCurrent(generation) {
		return
	}
	s.emit(session.Event{Type: session.EventError, Text: err.Error(), OccurredAt: time.Now().UTC()})
	s.stopResponse(generation, "")
}

func (s *stream) isCurrent(generation uint64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	return generation == s.generation && s.responseActive
}

func (s *stream) emitCurrent(ctx context.Context, generation uint64, event session.Event) {
	if s.isCurrent(generation) {
		select {
		case s.events <- event:
		case <-ctx.Done():
		case <-s.ctx.Done():
		}
	}
}

func (s *stream) emit(event session.Event) {
	select {
	case s.events <- event:
	case <-s.ctx.Done():
	}
}

func shouldFlushSpeechChunk(text string) bool {
	trimmed := strings.TrimSpace(text)
	if trimmed == "" {
		return false
	}
	last := trimmed[len(trimmed)-1]
	if last == '.' || last == '!' || last == '?' || last == ';' || last == ':' || last == '\n' {
		return true
	}
	return len(trimmed) >= 120 && len(text) > 0 && (text[len(text)-1] == ' ' || text[len(text)-1] == '\n')
}

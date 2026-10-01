// Package composable orchestrates provider-neutral speech-to-text, language
// model, and text-to-speech adapters as one realtime media session.
package composable

import (
	"context"
	"encoding/json"
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
	groqConfig := e.Groq
	groqConfig.Tools = make([]groq.Tool, 0, len(cfg.Tools))
	for _, tool := range cfg.Tools {
		groqConfig.Tools = append(groqConfig.Tools, groq.Tool{
			Type: "function",
			Function: groq.FunctionDefinition{
				Name:        tool.Name,
				Description: tool.Description,
				Parameters:  tool.Parameters,
			},
		})
	}
	s := &stream{
		ctx: streamCtx, cancel: cancel, transcriber: deepgramStream,
		generator: generator, synthesizer: synthesizer, groq: groqConfig,
		cartesia: cartesiaConfig, config: cfg,
		audio: make(chan session.AudioFrame), events: make(chan session.Event, 32), done: make(chan struct{}),
		pendingToolCalls: make(map[string]string),
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

	mu               sync.Mutex
	messages         []groq.Message
	generation       uint64
	responseCancel   context.CancelFunc
	responseActive   bool
	pendingToolCalls map[string]string
	responses        sync.WaitGroup
	closeOnce        sync.Once
	done             chan struct{}
}

func (s *stream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	return s.transcriber.SendAudio(ctx, frame)
}

func (s *stream) Interrupt(context.Context) error {
	s.cancelResponse()
	return nil
}

func (s *stream) SubmitToolResult(ctx context.Context, result session.ToolResult) error {
	if ctx == nil {
		return fmt.Errorf("tool result context is required")
	}
	s.mu.Lock()
	name, exists := s.pendingToolCalls[result.ToolCallID]
	if !exists {
		s.mu.Unlock()
		return fmt.Errorf("unknown pending tool call %q", result.ToolCallID)
	}
	if result.Name != "" && result.Name != name {
		s.mu.Unlock()
		return fmt.Errorf("tool result name does not match pending call")
	}
	delete(s.pendingToolCalls, result.ToolCallID)
	s.messages = append(s.messages, groq.Message{
		Role:       "tool",
		Content:    result.Content,
		ToolCallID: result.ToolCallID,
	})
	if len(s.pendingToolCalls) != 0 {
		s.mu.Unlock()
		return nil
	}
	s.generation++
	generation := s.generation
	responseCtx, cancel := context.WithCancel(s.ctx)
	s.responseCancel = cancel
	s.responseActive = true
	messages := append([]groq.Message(nil), s.messages...)
	s.responses.Add(1)
	s.mu.Unlock()
	s.emit(session.Event{Type: session.EventResponseStarted, OccurredAt: time.Now().UTC()})
	go s.generate(responseCtx, generation, messages)
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
				s.emit(session.Event{
					Type: session.EventError,
					Failure: &session.FailureEvent{
						Source: "deepgram", Message: event.Err.Error(), Terminal: true,
					},
					OccurredAt: time.Now().UTC(),
				})
				s.cancelResponse()
				s.responses.Wait()
				return
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
		s.emit(session.Event{
			Type:       session.EventTranscriptFinal,
			Transcript: &session.TranscriptEvent{Text: text},
			ProviderID: event.RequestID,
			OccurredAt: time.Now().UTC(),
		})
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
	toolCalls := make(map[int]*session.ToolCallEvent)
	toolOrder := make([]int, 0, 1)
	completionEvents := completion.Events()
	for {
		select {
		case event, ok := <-completionEvents:
			if !ok || event.Done {
				completeCalls := make([]*session.ToolCallEvent, 0, len(toolOrder))
				for _, index := range toolOrder {
					call := toolCalls[index]
					if call == nil || call.ID == "" || call.Name == "" || !json.Valid(call.Arguments) {
						s.emitCurrent(ctx, generation, session.Event{
							Type: session.EventError,
							Failure: &session.FailureEvent{
								Source: "groq", Code: "invalid_tool_call",
								Message: "Groq returned an incomplete tool call", Terminal: false,
							},
							OccurredAt: time.Now().UTC(),
						})
						continue
					}
					completeCalls = append(completeCalls, call)
					s.emitCurrent(ctx, generation, session.Event{
						Type:       session.EventToolCall,
						ToolCall:   call,
						OccurredAt: time.Now().UTC(),
					})
				}
				if len(completeCalls) > 0 {
					s.awaitToolResults(generation, completeCalls)
					return
				}
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
			if event.Usage != nil {
				s.emitCurrent(ctx, generation, session.Event{
					Type: session.EventUsage,
					Usage: &session.UsageEvent{
						InputTokens:  event.Usage.PromptTokens,
						OutputTokens: event.Usage.CompletionTokens,
						TotalTokens:  event.Usage.TotalTokens,
					},
					ProviderID: event.CompletionID,
					OccurredAt: time.Now().UTC(),
				})
			}
			if event.TextDelta != "" {
				text.WriteString(event.TextDelta)
				s.emitCurrent(ctx, generation, session.Event{
					Type:       session.EventResponseDelta,
					Response:   &session.ResponseEvent{Text: event.TextDelta},
					ProviderID: event.CompletionID,
					OccurredAt: time.Now().UTC(),
				})
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
			if event.ToolCallID != "" || event.ToolName != "" || len(event.ToolArguments) != 0 {
				call, exists := toolCalls[event.ToolIndex]
				if !exists {
					call = &session.ToolCallEvent{}
					toolCalls[event.ToolIndex] = call
					toolOrder = append(toolOrder, event.ToolIndex)
				}
				if event.ToolCallID != "" {
					call.ID = event.ToolCallID
				}
				if event.ToolName != "" {
					call.Name = event.ToolName
				}
				call.Arguments = append(call.Arguments, event.ToolArguments...)
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

func (s *stream) awaitToolResults(generation uint64, calls []*session.ToolCallEvent) {
	s.mu.Lock()
	if generation != s.generation || !s.responseActive {
		s.mu.Unlock()
		return
	}
	s.responseActive = false
	s.responseCancel = nil
	message := groq.Message{Role: "assistant"}
	for _, call := range calls {
		message.ToolCalls = append(message.ToolCalls, groq.MessageToolCall{
			ID:   call.ID,
			Type: "function",
			Function: groq.MessageFunctionCall{
				Name:      call.Name,
				Arguments: string(call.Arguments),
			},
		})
		s.pendingToolCalls[call.ID] = call.Name
	}
	s.messages = append(s.messages, message)
	s.mu.Unlock()
	s.emit(session.Event{Type: session.EventResponseStopped, OccurredAt: time.Now().UTC()})
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
	s.emit(session.Event{
		Type: session.EventError,
		Failure: &session.FailureEvent{
			Source: "composable", Message: err.Error(), Terminal: false,
		},
		OccurredAt: time.Now().UTC(),
	})
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

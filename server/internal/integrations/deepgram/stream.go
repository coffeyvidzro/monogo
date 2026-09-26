package deepgram

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type stream struct {
	ctx        context.Context
	cancel     context.CancelFunc
	connection *websocket.Conn
	format     session.AudioFormat
	events     chan Event
	writeMu    sync.Mutex
	closeOnce  sync.Once
}

func newStream(ctx context.Context, cancel context.CancelFunc, connection *websocket.Conn, format session.AudioFormat) *stream {
	return &stream{ctx: ctx, cancel: cancel, connection: connection, format: format, events: make(chan Event, 32)}
}

func (s *stream) SendAudio(ctx context.Context, frame session.AudioFrame) error {
	if err := frame.Validate(); err != nil {
		return err
	}
	if frame.Format != s.format {
		return fmt.Errorf("Deepgram audio format changed during stream")
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.connection.Write(ctx, websocket.MessageBinary, frame.Data)
}

func (s *stream) Finalize(ctx context.Context) error {
	return s.writeJSON(ctx, map[string]string{"type": "Finalize"})
}

func (s *stream) Events() <-chan Event { return s.events }

func (s *stream) Close(ctx context.Context) error {
	var closeErr error
	s.closeOnce.Do(func() {
		_ = s.writeJSON(ctx, map[string]string{"type": "CloseStream"})
		s.cancel()
		closeErr = s.connection.Close(websocket.StatusNormalClosure, "complete")
	})
	return closeErr
}

func (s *stream) writeJSON(ctx context.Context, value any) error {
	payload, err := json.Marshal(value)
	if err != nil {
		return err
	}
	s.writeMu.Lock()
	defer s.writeMu.Unlock()
	return s.connection.Write(ctx, websocket.MessageText, payload)
}

func (s *stream) readLoop() {
	defer close(s.events)
	for {
		messageType, payload, err := s.connection.Read(s.ctx)
		if err != nil {
			if s.ctx.Err() == nil && websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				s.emit(Event{Err: fmt.Errorf("read Deepgram stream: %w", err)})
			}
			return
		}
		if messageType != websocket.MessageText {
			s.emit(Event{Err: fmt.Errorf("Deepgram returned a non-text event")})
			return
		}
		var message Message
		if err := json.Unmarshal(payload, &message); err != nil {
			s.emit(Event{Err: fmt.Errorf("decode Deepgram event: %w", err)})
			return
		}
		if message.Type != "Results" || len(message.Channel.Alternatives) == 0 {
			continue
		}
		alternative := message.Channel.Alternatives[0]
		requestID := message.RequestID
		if requestID == "" {
			requestID = message.Metadata.RequestID
		}
		s.emit(Event{RequestID: requestID, Transcript: Transcript{
			Text: alternative.Transcript, Confidence: alternative.Confidence,
			IsFinal: message.IsFinal, SpeechFinal: message.SpeechFinal,
			Start: durationSeconds(message.Start), Duration: durationSeconds(message.Duration),
		}})
	}
}

func (s *stream) emit(event Event) {
	select {
	case s.events <- event:
	case <-s.ctx.Done():
	}
}

func durationSeconds(value float64) time.Duration { return time.Duration(value * float64(time.Second)) }

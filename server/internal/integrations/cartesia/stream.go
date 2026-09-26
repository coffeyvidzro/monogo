package cartesia

import (
	"context"
	"encoding/base64"
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
	contextID  string
	events     chan Event
	writeMu    sync.Mutex
	closeOnce  sync.Once
}

func newStream(
	ctx context.Context,
	cancel context.CancelFunc,
	connection *websocket.Conn,
	format session.AudioFormat,
	contextID string,
) *stream {
	return &stream{
		ctx:        ctx,
		cancel:     cancel,
		connection: connection,
		format:     format,
		contextID:  contextID,
		events:     make(chan Event, 32),
	}
}

func (s *stream) Events() <-chan Event {
	return s.events
}

func (s *stream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		closeCtx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = s.writeJSON(closeCtx, map[string]any{"context_id": s.contextID, "cancel": true})
		s.cancel()
		err = s.connection.CloseNow()
	})
	return err
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
	defer func() {
		_ = s.Close()
	}()
	for {
		kind, payload, err := s.connection.Read(s.ctx)
		if err != nil {
			if s.ctx.Err() == nil && websocket.CloseStatus(err) != websocket.StatusNormalClosure {
				s.emit(Event{Err: fmt.Errorf("read Cartesia stream: %w", err)})
			}
			return
		}
		if kind != websocket.MessageText {
			s.emit(Event{Err: fmt.Errorf("cartesia returned a non-text event")})
			return
		}
		var response Response
		if err := json.Unmarshal(payload, &response); err != nil {
			s.emit(Event{Err: fmt.Errorf("decode Cartesia event: %w", err)})
			return
		}
		if response.Type == "error" || response.StatusCode >= 400 {
			s.emit(Event{
				RequestID: response.RequestID,
				Err:       fmt.Errorf("cartesia %s: %s", response.ErrorCode, response.Message),
			})
			return
		}
		switch response.Type {
		case "chunk":
			audio, err := base64.StdEncoding.DecodeString(response.Data)
			if err != nil {
				s.emit(Event{Err: fmt.Errorf("decode Cartesia audio: %w", err)})
				return
			}
			frame := session.AudioFrame{
				Data:       audio,
				Format:     s.format,
				CapturedAt: time.Now().UTC(),
			}
			if err := frame.Validate(); err != nil {
				s.emit(Event{Err: err})
				return
			}
			s.emit(Event{Audio: frame, RequestID: response.RequestID})
		case "done":
			s.emit(Event{Done: true, RequestID: response.RequestID})
			return
		}
	}
}
func (s *stream) emit(event Event) {
	select {
	case s.events <- event:
	case <-s.ctx.Done():
	}
}

package groq

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type stream struct {
	ctx       context.Context
	cancel    context.CancelFunc
	body      io.ReadCloser
	events    chan Event
	done      chan struct{}
	closeOnce sync.Once
}

func newStream(parent context.Context, body io.ReadCloser) *stream {
	ctx, cancel := context.WithCancel(parent)
	return &stream{
		ctx: ctx, cancel: cancel, body: body,
		events: make(chan Event, 32), done: make(chan struct{}),
	}
}

func (s *stream) Events() <-chan Event {
	return s.events
}

func (s *stream) Close() error {
	var err error
	s.closeOnce.Do(func() {
		s.cancel()
		err = s.body.Close()
	})
	return err
}

func (s *stream) readLoop() {
	defer close(s.done)
	defer close(s.events)
	defer func() {
		_ = s.Close()
	}()
	scanner := bufio.NewScanner(s.body)
	scanner.Buffer(make([]byte, 64<<10), 1<<20)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, ":") || !strings.HasPrefix(line, "data:") {
			continue
		}
		data := strings.TrimSpace(strings.TrimPrefix(line, "data:"))
		if data == "[DONE]" {
			if !s.emit(Event{Done: true}) {
			return
		}
			return
		}
		var chunk CompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			if !s.emit(Event{Err: fmt.Errorf("decode Groq stream chunk: %w", err)}) {
				return
			}
			return
		}
		if len(chunk.Choices) == 0 {
			if !s.emit(Event{CompletionID: chunk.ID, Usage: chunk.Usage}) {
				return
			}
			continue
		}
		for _, choice := range chunk.Choices {
			base := Event{
				CompletionID: chunk.ID,
				TextDelta:    choice.Delta.Content,
				FinishReason: choice.FinishReason,
				Usage:        chunk.Usage,
			}
			if len(choice.Delta.ToolCalls) == 0 {
				if !s.emit(base) {
					return
				}
				continue
			}
			if base.TextDelta != "" {
				if !s.emit(base) {
					return
				}
			}
			for _, call := range choice.Delta.ToolCalls {
				if !s.emit(Event{
					CompletionID:  chunk.ID,
					ToolCallID:    call.ID,
					ToolIndex:     call.Index,
					ToolName:      call.Function.Name,
					ToolArguments: []byte(call.Function.Arguments),
					FinishReason:  choice.FinishReason,
				}) {
					return
				}
			}
		}
	}
	if err := scanner.Err(); err != nil && s.ctx.Err() == nil {
		s.emit(Event{Err: fmt.Errorf("read Groq stream: %w", err)})
	}
}

func (s *stream) emit(event Event) bool {
	select {
	case s.events <- event:
		return true
	case <-s.ctx.Done():
		return false
	}
}

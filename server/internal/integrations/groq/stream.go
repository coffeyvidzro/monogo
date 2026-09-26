package groq

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"strings"
	"sync"
)

type stream struct {
	body      io.ReadCloser
	events    chan Event
	closeOnce sync.Once
}

func newStream(body io.ReadCloser) *stream {
	return &stream{
		body:   body,
		events: make(chan Event, 32),
	}
}

func (s *stream) Events() <-chan Event {
	return s.events
}

func (s *stream) Close() error {
	var err error
	s.closeOnce.Do(func() { err = s.body.Close() })
	return err
}

func (s *stream) readLoop() {
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
			s.events <- Event{Done: true}
			return
		}
		var chunk CompletionChunk
		if err := json.Unmarshal([]byte(data), &chunk); err != nil {
			s.events <- Event{Err: fmt.Errorf("decode Groq stream chunk: %w", err)}
			return
		}
		if len(chunk.Choices) == 0 {
			s.events <- Event{CompletionID: chunk.ID, Usage: chunk.Usage}
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
				s.events <- base
				continue
			}
			if base.TextDelta != "" {
				s.events <- base
			}
			for _, call := range choice.Delta.ToolCalls {
				s.events <- Event{
					CompletionID:  chunk.ID,
					ToolCallID:    call.ID,
					ToolIndex:     call.Index,
					ToolName:      call.Function.Name,
					ToolArguments: []byte(call.Function.Arguments),
					FinishReason:  choice.FinishReason,
				}
			}
		}
	}
	if err := scanner.Err(); err != nil {
		s.events <- Event{Err: fmt.Errorf("read Groq stream: %w", err)}
	}
}

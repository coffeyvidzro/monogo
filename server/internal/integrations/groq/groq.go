// Package groq defines the streaming language-model integration boundary.
package groq

import "context"

type Config struct {
	APIKey string
	Model  string
}

type Message struct {
	Role    string
	Content string
}

type Event struct {
	TextDelta     string
	ToolName      string
	ToolArguments []byte
	Done          bool
	Err           error
}

type Stream interface {
	Events() <-chan Event
	Close() error
}

type Generator interface {
	Generate(context.Context, Config, []Message) (Stream, error)
}

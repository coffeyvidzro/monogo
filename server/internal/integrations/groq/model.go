// Package groq implements Groq's OpenAI-compatible chat completion stream.
package groq

import (
	"context"
	"encoding/json"
)

const (
	DefaultEndpoint = "https://api.groq.com/openai/v1/chat/completions"
	DefaultModel    = "qwen/qwen3.8-27b"
)

type Config struct {
	APIKey              string
	Endpoint            string
	Model               string
	Temperature         *float64
	MaxCompletionTokens int
	Tools               []Tool
}

type Message struct {
	Role    string `json:"role"`
	Content string `json:"content"`
}

type Tool struct {
	Type     string             `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type CompletionRequest struct {
	Model               string    `json:"model"`
	Messages            []Message `json:"messages"`
	Stream              bool      `json:"stream"`
	Temperature         *float64  `json:"temperature,omitempty"`
	MaxCompletionTokens int       `json:"max_completion_tokens,omitempty"`
	Tools               []Tool    `json:"tools,omitempty"`
}

type CompletionChunk struct {
	ID      string   `json:"id"`
	Choices []Choice `json:"choices"`
	Usage   *Usage   `json:"usage,omitempty"`
}

type Choice struct {
	Index        int    `json:"index"`
	Delta        Delta  `json:"delta"`
	FinishReason string `json:"finish_reason"`
}

type Delta struct {
	Content   string     `json:"content"`
	ToolCalls []ToolCall `json:"tool_calls"`
}

type ToolCall struct {
	Index    int          `json:"index"`
	ID       string       `json:"id"`
	Type     string       `json:"type"`
	Function FunctionCall `json:"function"`
}

type FunctionCall struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`
}

type Event struct {
	CompletionID  string
	TextDelta     string
	ToolCallID    string
	ToolIndex     int
	ToolName      string
	ToolArguments []byte
	FinishReason  string
	Usage         *Usage
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

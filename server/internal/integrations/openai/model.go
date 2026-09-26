// Package openai implements the server-to-server OpenAI Realtime WebSocket API.
package openai

import "encoding/json"

const (
	DefaultEndpoint = "wss://api.openai.com/v1/realtime"
	DefaultModel    = "gpt-realtime-2.1"
)

type Config struct {
	APIKey           string
	Endpoint         string
	Model            string
	Voice            string
	SafetyIdentifier string
	Tools            []Tool
}

type Tool struct {
	Type        string          `json:"type"`
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters"`
}

type ClientEvent struct {
	Type       string         `json:"type"`
	EventID    string         `json:"event_id,omitempty"`
	Session    *SessionUpdate `json:"session,omitempty"`
	Audio      string         `json:"audio,omitempty"`
	ResponseID string         `json:"response_id,omitempty"`
}

type SessionUpdate struct {
	Type             string      `json:"type,omitempty"`
	Instructions     string      `json:"instructions,omitempty"`
	OutputModalities []string    `json:"output_modalities,omitempty"`
	Audio            AudioConfig `json:"audio"`
	Tools            []Tool      `json:"tools,omitempty"`
}

type AudioConfig struct {
	Input  AudioInput  `json:"input"`
	Output AudioOutput `json:"output"`
}

type AudioInput struct {
	Format AudioFormat `json:"format"`
}
type AudioOutput struct {
	Format AudioFormat `json:"format"`
	Voice  string      `json:"voice,omitempty"`
}
type AudioFormat struct {
	Type string `json:"type"`
	Rate int    `json:"rate"`
}

type ServerEvent struct {
	Type       string    `json:"type"`
	EventID    string    `json:"event_id"`
	ResponseID string    `json:"response_id"`
	ItemID     string    `json:"item_id"`
	CallID     string    `json:"call_id"`
	Name       string    `json:"name"`
	Delta      string    `json:"delta"`
	Transcript string    `json:"transcript"`
	Arguments  string    `json:"arguments"`
	Error      *APIError `json:"error"`
	Response   *Response `json:"response"`
}

type APIError struct {
	Type    string `json:"type"`
	Code    string `json:"code"`
	Message string `json:"message"`
}
type Response struct {
	ID     string          `json:"id"`
	Status string          `json:"status"`
	Usage  json.RawMessage `json:"usage"`
}

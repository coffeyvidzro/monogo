// Package openai defines the speech-to-speech realtime integration boundary.
package openai

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type Config struct {
	APIKey string
	Model  string
	Voice  string
}

// Client starts OpenAI realtime streams. Provider events are normalized by the
// returned session.Stream before reaching media orchestration code.
type Client interface {
	Start(context.Context, Config, session.Config) (session.Stream, error)
}

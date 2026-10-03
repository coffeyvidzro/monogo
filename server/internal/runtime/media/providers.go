package media

import (
	"github.com/coffeyvidzro/monogo/internal/integrations/cartesia"
	"github.com/coffeyvidzro/monogo/internal/integrations/deepgram"
	"github.com/coffeyvidzro/monogo/internal/integrations/groq"
	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

func builtInProviderRegistry(cfg Config) (*providersdk.Registry, error) {
	return providersdk.NewRegistry(
		deepgram.Provider{},
		groq.Provider{},
		cartesia.Provider{},
		openai.Provider{
			Config: openai.Config{
				APIKey:   cfg.OpenAIAPIKey,
				Endpoint: cfg.OpenAIEndpoint,
			},
		},
	)
}

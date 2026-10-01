package media

import (
	"github.com/coffeyvidzro/monogo/internal/integrations/cartesia"
	"github.com/coffeyvidzro/monogo/internal/integrations/deepgram"
	"github.com/coffeyvidzro/monogo/internal/integrations/groq"
	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

func builtInProviderRegistry() (*providersdk.Registry, error) {
	return providersdk.NewRegistry(
		deepgram.Provider{}.Descriptor(),
		groq.Provider{}.Descriptor(),
		cartesia.Provider{}.Descriptor(),
		openai.Provider{}.Descriptor(),
	)
}

package openai

import (
	"context"
	"strings"
	"testing"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	providersdk "github.com/coffeyvidzro/monogo/internal/providers"
)

func TestProviderDoesNotFallBackToConfiguredAPIKey(t *testing.T) {
	provider := Provider{
		Config: Config{
			APIKey: "deployment-global-secret",
		},
	}

	_, err := provider.StartRealtime(
		context.Background(),
		providersdk.Runtime{},
		session.Config{},
	)
	if err == nil || !strings.Contains(err.Error(), "OpenAI API key is required") {
		t.Fatalf("StartRealtime() error = %v", err)
	}
}

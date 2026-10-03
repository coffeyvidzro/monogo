package media

import (
	"context"
	"strings"
	"testing"
	"time"

	"github.com/coffeyvidzro/monogo/internal/media/session"
)

func TestRunStopsWithContext(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- RunWithConfig(ctx, validRuntimeConfig()) }()
	time.Sleep(20 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("RunWithConfig() error = %v", err)
		}
	case <-time.After(time.Second):
		t.Fatal("RunWithConfig() did not stop")
	}
}

func TestRunRejectsNilContext(t *testing.T) {
	var nilContext context.Context
	if err := Run(nilContext); err == nil {
		t.Fatal("Run() error = nil")
	}
}

func TestMediaEnginesAlwaysRegisterTenantCapableEngines(t *testing.T) {
	cfg := validRuntimeConfig()
	registry, err := builtInProviderRegistry(cfg)
	if err != nil {
		t.Fatalf("builtInProviderRegistry() error = %v", err)
	}
	engines := mediaEngines(registry)

	for _, engine := range []session.Engine{
		session.EngineEcho,
		session.EngineIntegrated,
		session.EngineComposable,
	} {
		if _, exists := engines[engine]; !exists {
			t.Fatalf("%s engine not registered", engine)
		}
	}
}

func TestConfigAcceptsNoGlobalProviderCredentials(t *testing.T) {
	cfg := validRuntimeConfig()

	if err := cfg.Validate(); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
}

func validRuntimeConfig() Config {
	return Config{
		ListenAddress: "127.0.0.1:0", PublicWebSocket: "ws://media:8090/v1/audio-forks",
		TokenSecret: strings.Repeat("s", 32), ControlToken: strings.Repeat("c", 32),
		TokenTTL: time.Minute, AttachTimeout: time.Minute, MaxSessions: 2, ReadLimit: 65536,
		HandshakeTimeout: time.Second, DrainTimeout: time.Second,
	}
}

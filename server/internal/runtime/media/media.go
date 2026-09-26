// Package media assembles and runs the realtime media plane.
package media

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/openai"
	"github.com/coffeyvidzro/monogo/internal/media/engine/echo"
	"github.com/coffeyvidzro/monogo/internal/media/engine/integrated"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/media/transport"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

func Run(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("media runtime context is required")
	}
	cfg, err := loadConfig()
	if err != nil {
		return fmt.Errorf("load media configuration: %w", err)
	}
	return RunWithConfig(ctx, cfg)
}

func RunWithConfig(ctx context.Context, cfg Config) error {
	if ctx == nil {
		return fmt.Errorf("media runtime context is required")
	}
	if err := cfg.Validate(); err != nil {
		return err
	}
	logger := logging.New().With("process", "media")
	manager, err := session.NewManager(cfg.MaxSessions, cfg.TokenTTL, mediaEngines(cfg))
	if err != nil {
		return fmt.Errorf("initialize media session manager: %w", err)
	}
	tokens, err := transport.NewTokenService(cfg.TokenSecret)
	if err != nil {
		return fmt.Errorf("initialize media tokens: %w", err)
	}
	websockets, err := transport.NewWebSocketHandler(tokens, manager, transport.WebSocketConfig{
		HandshakeTimeout: cfg.HandshakeTimeout, ReadLimit: cfg.ReadLimit,
	})
	if err != nil {
		return fmt.Errorf("initialize media WebSocket: %w", err)
	}
	handler := newHandler(cfg, manager, tokens, websockets)
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", cfg.ListenAddress)
	if err != nil {
		return fmt.Errorf("listen for media: %w", err)
	}
	server := &http.Server{Handler: handler, ReadHeaderTimeout: 5 * time.Second}
	serverErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()
	logger.Info(ctx, "media worker started", "address", listener.Addr().String())

	var result error
	select {
	case <-ctx.Done():
	case result = <-serverErr:
		if result == nil && ctx.Err() == nil {
			result = fmt.Errorf("media HTTP server exited unexpectedly")
		}
	}

	handler.draining.Store(true)
	drainCtx, cancel := context.WithTimeout(context.Background(), cfg.DrainTimeout)
	defer cancel()
	if err := manager.Drain(drainCtx); err != nil && result == nil {
		result = fmt.Errorf("drain media sessions: %w", err)
	}
	if err := server.Shutdown(drainCtx); err != nil && result == nil {
		result = fmt.Errorf("shutdown media HTTP server: %w", err)
	}
	logger.Info(context.Background(), "media worker stopped")
	return result
}


func mediaEngines(cfg Config) map[session.Engine]session.Starter {
	engines := map[session.Engine]session.Starter{
		session.EngineEcho: echo.Engine{},
	}
	if cfg.OpenAIAPIKey != "" {
		engines[session.EngineIntegrated] = integrated.Engine{
			Client: openai.NewClient(nil),
			Config: openai.Config{APIKey: cfg.OpenAIAPIKey},
		}
	}
	return engines
}

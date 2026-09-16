package server

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

func Run(ctx context.Context) error {
	logger := logging.New().With("process", "server")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	modules, err := newModules(ctx, cfg)
	if err != nil {
		return err
	}
	defer modules.close(logger)

	httpServer := &http.Server{
		Addr:              ":8080",
		Handler:           newRouter(cfg, logger, modules),
		ReadHeaderTimeout: 10 * time.Second,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(ctx, "server listening", "address", httpServer.Addr)
		err := httpServer.ListenAndServe()
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	select {
	case err := <-serverErr:
		if err != nil {
			return fmt.Errorf("serve HTTP: %w", err)
		}
		return nil
	case <-ctx.Done():
	}

	logger.Info(context.Background(), "server shutting down")
	shutdownCtx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

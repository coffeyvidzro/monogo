package main

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/go-chi/chi/v5"

	"github.com/coffeyvidzro/monogo/internal/integrations/postgres"
	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/platform/config"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

const (
	httpAddress       = ":8080"
	probeTimeout      = 2 * time.Second
	shutdownTimeout   = 15 * time.Second
	readHeaderTimeout = 10 * time.Second
)

func main() {
	if err := run(); err != nil {
		logging.New().Error(context.Background(), "server failed", "error", err)
		os.Exit(1)
	}
}

func run() error {
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	logger := logging.New().With("process", "server")

	cfg, err := config.Load()
	if err != nil {
		return fmt.Errorf("load configuration: %w", err)
	}

	postgresClient, err := postgres.New(ctx, postgres.DefaultConfig(cfg.DatabaseURL))
	if err != nil {
		return fmt.Errorf("initialize PostgreSQL: %w", err)
	}
	defer postgresClient.Close()

	redisClient, err := redisintegration.New(ctx, redisintegration.DefaultConfig(cfg.RedisURL))
	if err != nil {
		return fmt.Errorf("initialize Redis: %w", err)
	}
	defer func() {
		if err := redisClient.Close(); err != nil {
			logger.Warn(context.Background(), "close Redis", "error", err)
		}
	}()

	router := chi.NewRouter()
	registerHealthRoutes(router, postgresClient, redisClient)

	httpServer := &http.Server{
		Addr:              httpAddress,
		Handler:           router,
		ReadHeaderTimeout: readHeaderTimeout,
	}

	serverErr := make(chan error, 1)
	go func() {
		logger.Info(ctx, "server listening", "address", httpAddress)
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
	shutdownCtx, cancel := context.WithTimeout(context.Background(), shutdownTimeout)
	defer cancel()
	if err := httpServer.Shutdown(shutdownCtx); err != nil {
		return fmt.Errorf("shutdown HTTP server: %w", err)
	}
	if err := <-serverErr; err != nil {
		return fmt.Errorf("serve HTTP: %w", err)
	}
	return nil
}

func registerHealthRoutes(router chi.Router, postgresClient *postgres.Client, redisClient *redisintegration.Client) {
	router.Get("/healthz", func(w http.ResponseWriter, _ *http.Request) {
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ok\n"))
	})

	router.Get("/readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), probeTimeout)
		defer cancel()

		if err := postgresClient.Ping(ctx); err != nil {
			http.Error(w, "postgres unavailable", http.StatusServiceUnavailable)
			return
		}
		if err := redisClient.Ping(ctx); err != nil {
			http.Error(w, "redis unavailable", http.StatusServiceUnavailable)
			return
		}

		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
}

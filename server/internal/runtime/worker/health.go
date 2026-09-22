package worker

import (
	"context"
	"errors"
	"fmt"
	"net"
	"net/http"
	"time"

	"github.com/coffeyvidzro/monogo/internal/platform/logging"
)

func (m *modules) ready(ctx context.Context) error {
	if err := m.postgres.Ping(ctx); err != nil {
		return fmt.Errorf("postgres: %w", err)
	}
	if err := m.redis.Ping(ctx); err != nil {
		return fmt.Errorf("redis: %w", err)
	}
	if err := m.nats.Ping(ctx); err != nil {
		return fmt.Errorf("nats: %w", err)
	}
	if err := m.freeSwitch.HealthCheck(ctx); err != nil {
		return fmt.Errorf("freeswitch: %w", err)
	}
	return nil
}

// Worker readiness is private to its container. The listener starts only after
// the FreeSWITCH lifecycle subscription has been acknowledged.
func runWorkloadsWithReadiness(ctx context.Context, logger *logging.Logger, modules *modules) error {
	listener, err := (&net.ListenConfig{}).Listen(ctx, "tcp", "127.0.0.1:8081")
	if err != nil {
		return fmt.Errorf("listen for worker readiness: %w", err)
	}

	runCtx, cancel := context.WithCancel(ctx)
	defer cancel()

	server := &http.Server{
		Handler:           workerReadinessHandler(modules.ready),
		ReadHeaderTimeout: 5 * time.Second,
	}
	serverErr := make(chan error, 1)
	go func() {
		err := server.Serve(listener)
		if errors.Is(err, http.ErrServerClosed) {
			err = nil
		}
		serverErr <- err
	}()

	workErr := make(chan error, 1)
	go func() { workErr <- runWorkloads(runCtx, logger, modules) }()

	var result error
	select {
	case <-ctx.Done():
	case err := <-workErr:
		result = err
		if err == nil && ctx.Err() == nil {
			result = fmt.Errorf("worker workloads exited unexpectedly")
		}
	case err := <-serverErr:
		result = err
		if err == nil && ctx.Err() == nil {
			result = fmt.Errorf("worker readiness server exited unexpectedly")
		}
	}

	cancel()
	shutdownCtx, stop := context.WithTimeout(context.Background(), 5*time.Second)
	defer stop()
	if err := server.Shutdown(shutdownCtx); err != nil {
		if result == nil {
			result = fmt.Errorf("shutdown worker readiness: %w", err)
		}
	}
	return result
}

func workerReadinessHandler(ready func(context.Context) error) http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("GET /readyz", func(w http.ResponseWriter, r *http.Request) {
		ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
		defer cancel()
		if err := ready(ctx); err != nil {
			http.Error(w, "worker dependencies unavailable", http.StatusServiceUnavailable)
			return
		}
		w.Header().Set("Content-Type", "text/plain; charset=utf-8")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("ready\n"))
	})
	return mux
}

package media

import (
	"context"
	"fmt"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/platform/logging"
	"github.com/coffeyvidzro/monogo/internal/runtime/medianodes"
)

const mediaNodeHeartbeatInterval = 5 * time.Second

type nodeRegistration struct {
	redis    *redisintegration.Client
	registry *medianodes.Registry
	nodeID   string
	cancel   context.CancelFunc
	done     chan struct{}
}

func startNodeRegistration(
	ctx context.Context,
	cfg Config,
	manager *session.Manager,
	logger *logging.Logger,
) (*nodeRegistration, error) {
	if cfg.RedisURL == "" {
		return &nodeRegistration{}, nil
	}

	client, err := redisintegration.New(
		ctx,
		redisintegration.DefaultConfig(cfg.RedisURL),
	)
	if err != nil {
		return nil, fmt.Errorf("initialize media Redis: %w", err)
	}

	registry := medianodes.NewRegistry(client)
	registrationCtx, cancel := context.WithCancel(context.Background())
	registration := &nodeRegistration{
		redis:    client,
		registry: registry,
		nodeID:   cfg.NodeID,
		cancel:   cancel,
		done:     make(chan struct{}),
	}

	heartbeat := func(draining bool) error {
		if err := registry.Heartbeat(registrationCtx, medianodes.Node{
			ID:         cfg.NodeID,
			ControlURL: cfg.ControlURL,
			AudioURL:   cfg.PublicWebSocket,
			Capacity:   cfg.MaxSessions,
			Active:     manager.Active(),
			Draining:   draining,
		}); err != nil {
			return err
		}
		for _, sessionID := range manager.SessionIDs() {
			if err := registry.Refresh(registrationCtx, cfg.NodeID, sessionID); err != nil && logger != nil {
				logger.Warn(
					registrationCtx,
					"refresh media session ownership",
					"node_id", cfg.NodeID,
					"session_id", sessionID,
					"error", err,
				)
			}
		}
		return nil
	}

	if err := heartbeat(false); err != nil {
		cancel()
		_ = client.Close()
		return nil, fmt.Errorf("heartbeat media node: %w", err)
	}

	go func() {
		defer close(registration.done)
		ticker := time.NewTicker(mediaNodeHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-registrationCtx.Done():
				return
			case <-ticker.C:
				if err := heartbeat(false); err != nil && logger != nil {
					logger.Warn(
						registrationCtx,
						"heartbeat media node",
						"node_id", cfg.NodeID,
						"error", err,
					)
				}
			}
		}
	}()

	return registration, nil
}

func (r *nodeRegistration) beginDrain(
	ctx context.Context,
	cfg Config,
	manager *session.Manager,
) {
	if r == nil || r.registry == nil {
		return
	}
	r.cancel()
	select {
	case <-r.done:
	case <-ctx.Done():
		return
	}
	_ = r.registry.Heartbeat(ctx, medianodes.Node{
		ID:         cfg.NodeID,
		ControlURL: cfg.ControlURL,
		AudioURL:   cfg.PublicWebSocket,
		Capacity:   cfg.MaxSessions,
		Active:     manager.Active(),
		Draining:   true,
	})
}

func (r *nodeRegistration) close(ctx context.Context) {
	if r == nil || r.registry == nil {
		return
	}
	r.cancel()
	select {
	case <-r.done:
	case <-ctx.Done():
	}
	_ = r.registry.RemoveNode(context.Background(), r.nodeID)
	_ = r.redis.Close()
}

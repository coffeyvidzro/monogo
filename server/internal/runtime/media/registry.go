package media

import (
	"context"
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
	cancel   context.CancelFunc
	done     chan struct{}
	nodeID   string
}

func startNodeRegistration(
	ctx context.Context,
	cfg Config,
	manager *session.Manager,
	logger *logging.Logger,
) (*nodeRegistration, error) {
	if cfg.RedisURL == "" {
		return nil, nil
	}

	client, err := redisintegration.New(
		ctx,
		redisintegration.DefaultConfig(cfg.RedisURL),
	)
	if err != nil {
		return nil, err
	}

	registry := medianodes.NewRegistry(client)
	registrationCtx, cancel := context.WithCancel(context.Background())
	registration := &nodeRegistration{
		redis:    client,
		registry: registry,
		cancel:   cancel,
		done:     make(chan struct{}),
		nodeID:   cfg.NodeID,
	}

	heartbeat := func(draining bool) {
		err := registry.Heartbeat(registrationCtx, medianodes.Node{
			ID:         cfg.NodeID,
			ControlURL: cfg.ControlURL,
			AudioURL:   cfg.PublicWebSocket,
			Capacity:   cfg.MaxSessions,
			Active:     manager.Active(),
			Draining:   draining,
		})
		if err != nil && logger != nil {
			logger.Warn(
				context.Background(),
				"heartbeat media node",
				"node_id", cfg.NodeID,
				"error", err,
			)
		}
		for _, sessionID := range manager.SessionIDs() {
			_ = registry.Refresh(registrationCtx, cfg.NodeID, sessionID)
		}
	}

	heartbeat(false)
	go func() {
		defer close(registration.done)
		ticker := time.NewTicker(mediaNodeHeartbeatInterval)
		defer ticker.Stop()
		for {
			select {
			case <-registrationCtx.Done():
				return
			case <-ticker.C:
				heartbeat(false)
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
	if r == nil {
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
	if r == nil {
		return
	}
	r.cancel()
	<-r.done
	_ = r.registry.RemoveNode(ctx, r.nodeID)
	_ = r.redis.Close()
}

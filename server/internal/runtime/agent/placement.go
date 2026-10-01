package agent

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/media/session"
	"github.com/coffeyvidzro/monogo/internal/runtime/medianodes"
	"github.com/google/uuid"
)

func (r *Runtime) createMediaSession(
	ctx context.Context,
	cfg session.Config,
) (mediaSessionEndpoints, error) {
	if r.mediaNodes == nil {
		return r.media.CreateSession(ctx, cfg)
	}

	node, err := r.mediaNodes.Place(ctx, cfg.ID)
	if err != nil {
		if r.logger != nil {
			r.logger.Warn(
				ctx,
				"media node placement unavailable; using static fallback",
				"voice_agent_session_id", cfg.ID,
				"error", err,
			)
		}
		return r.media.CreateSession(ctx, cfg)
	}

	endpoints, err := r.media.CreateSessionAt(ctx, node.ControlURL, cfg)
	if err != nil {
		_ = r.mediaNodes.Release(context.Background(), cfg.ID)
		return mediaSessionEndpoints{}, err
	}

	r.mu.Lock()
	r.sessionNodes[cfg.ID] = node
	r.mu.Unlock()
	return endpoints, nil
}

func (r *Runtime) mediaNode(
	ctx context.Context,
	sessionID uuid.UUID,
) (medianodes.Node, bool) {
	r.mu.Lock()
	node, ok := r.sessionNodes[sessionID]
	r.mu.Unlock()
	if ok {
		return node, true
	}
	if r.mediaNodes == nil {
		return medianodes.Node{}, false
	}

	node, err := r.mediaNodes.Owner(ctx, sessionID)
	if err != nil {
		return medianodes.Node{}, false
	}
	r.mu.Lock()
	r.sessionNodes[sessionID] = node
	r.mu.Unlock()
	return node, true
}

func (r *Runtime) releaseMediaNode(
	ctx context.Context,
	sessionID uuid.UUID,
) {
	r.mu.Lock()
	delete(r.sessionNodes, sessionID)
	r.mu.Unlock()

	if r.mediaNodes != nil {
		_ = r.mediaNodes.Release(ctx, sessionID)
	}
}

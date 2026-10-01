package medianodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	nodePrefix     = "runtime:media:node:"
	drainingPrefix = "runtime:media:draining:"
	leasesPrefix   = "runtime:media:leases:"
	ownerPrefix    = "runtime:media:session:"
	heartbeatTTL   = 15 * time.Second
	leaseTTL       = 45 * time.Second
)

type Node struct {
	ID         string    `json:"id"`
	ControlURL string    `json:"control_url"`
	AudioURL   string    `json:"audio_url"`
	Capacity   int       `json:"capacity"`
	Active     int       `json:"active"`
	Draining   bool      `json:"draining"`
	UpdatedAt  time.Time `json:"updated_at"`
}

type Registry struct {
	redis *redisintegration.Client
}

func NewRegistry(redis *redisintegration.Client) *Registry {
	if redis == nil {
		panic("medianodes: Redis client is required")
	}
	return &Registry{redis: redis}
}

func (r *Registry) Heartbeat(ctx context.Context, node Node) error {
	node.ID = strings.TrimSpace(node.ID)
	node.ControlURL = strings.TrimRight(strings.TrimSpace(node.ControlURL), "/")
	node.AudioURL = strings.TrimSpace(node.AudioURL)
	if node.ID == "" || node.ControlURL == "" || node.AudioURL == "" || node.Capacity <= 0 || node.Active < 0 {
		return fmt.Errorf("valid media node id, URLs, capacity, and active count are required")
	}
	node.UpdatedAt = time.Now().UTC()
	if err := r.redis.SetJSON(ctx, nodeKey(node.ID), node, heartbeatTTL); err != nil {
		return fmt.Errorf("heartbeat media node: %w", err)
	}
	if node.Draining {
		if err := r.redis.Set(ctx, drainingKey(node.ID), "1", heartbeatTTL); err != nil {
			return fmt.Errorf("mark media node draining: %w", err)
		}
		return nil
	}
	if err := r.redis.Delete(ctx, drainingKey(node.ID)); err != nil {
		return fmt.Errorf("clear media node draining: %w", err)
	}
	return nil
}

func (r *Registry) RemoveNode(ctx context.Context, nodeID string) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("media node id is required")
	}
	return r.redis.Delete(ctx, nodeKey(nodeID), drainingKey(nodeID))
}

func (r *Registry) Place(ctx context.Context, sessionID uuid.UUID) (Node, error) {
	if sessionID == uuid.Nil {
		return Node{}, fmt.Errorf("media session id is required")
	}

	nodes, err := r.Nodes(ctx)
	if err != nil {
		return Node{}, err
	}
	sort.Slice(nodes, func(i, j int) bool {
		leftFree := nodes[i].Capacity - nodes[i].Active
		rightFree := nodes[j].Capacity - nodes[j].Active
		if leftFree == rightFree {
			return nodes[i].ID < nodes[j].ID
		}
		return leftFree > rightFree
	})

	for _, node := range nodes {
		if node.Draining || node.Active >= node.Capacity {
			continue
		}
		ok, _, err := r.redis.AcquireMediaLease(
			ctx,
			nodeKey(node.ID),
			drainingKey(node.ID),
			leasesKey(node.ID),
			ownerKey(sessionID),
			node.ID,
			sessionID.String(),
			int64(node.Capacity),
			leaseTTL,
		)
		if err != nil {
			return Node{}, err
		}
		if ok {
			return node, nil
		}
	}
	return Node{}, fmt.Errorf("no healthy media node has available capacity")
}

func (r *Registry) Owner(ctx context.Context, sessionID uuid.UUID) (Node, error) {
	if sessionID == uuid.Nil {
		return Node{}, fmt.Errorf("media session id is required")
	}
	nodeID, err := r.redis.Get(ctx, ownerKey(sessionID))
	if errors.Is(err, redisv9.Nil) {
		return Node{}, fmt.Errorf("media session ownership not found")
	}
	if err != nil {
		return Node{}, err
	}
	var node Node
	if err := r.redis.GetJSON(ctx, nodeKey(nodeID), &node); err != nil {
		if errors.Is(err, redisv9.Nil) {
			return Node{}, fmt.Errorf("media session owner node is unavailable")
		}
		return Node{}, err
	}
	return node, nil
}

func (r *Registry) Release(ctx context.Context, sessionID uuid.UUID) error {
	if sessionID == uuid.Nil {
		return fmt.Errorf("media session id is required")
	}
	nodeID, err := r.redis.Get(ctx, ownerKey(sessionID))
	if errors.Is(err, redisv9.Nil) {
		return nil
	}
	if err != nil {
		return err
	}
	return r.redis.ReleaseMediaLease(
		ctx,
		leasesKey(nodeID),
		ownerKey(sessionID),
		nodeID,
		sessionID.String(),
	)
}

func (r *Registry) Refresh(ctx context.Context, nodeID string, sessionID uuid.UUID) error {
	return r.redis.RefreshMediaLease(
		ctx,
		nodeKey(nodeID),
		leasesKey(nodeID),
		ownerKey(sessionID),
		nodeID,
		sessionID.String(),
		leaseTTL,
	)
}

func (r *Registry) Nodes(ctx context.Context) ([]Node, error) {
	keys, err := r.redis.ScanKeys(ctx, nodePrefix+"*")
	if err != nil {
		return nil, err
	}
	result := make([]Node, 0, len(keys))
	for _, key := range keys {
		var node Node
		if err := r.redis.GetJSON(ctx, key, &node); err != nil {
			if errors.Is(err, redisv9.Nil) {
				continue
			}
			return nil, err
		}
		active, countErr := r.redis.CountMediaLeases(ctx, leasesKey(node.ID))
		if countErr != nil {
			return nil, countErr
		}
		node.Active = int(active)
		node.Draining = node.Draining || r.isDraining(ctx, node.ID)
		result = append(result, node)
	}
	return result, nil
}

func (r *Registry) isDraining(ctx context.Context, nodeID string) bool {
	exists, err := r.redis.Exists(ctx, drainingKey(nodeID))
	return err == nil && exists
}

func nodeKey(nodeID string) string {
	return nodePrefix + nodeID
}

func drainingKey(nodeID string) string {
	return drainingPrefix + nodeID
}

func leasesKey(nodeID string) string {
	return leasesPrefix + nodeID
}

func ownerKey(sessionID uuid.UUID) string {
	return ownerPrefix + sessionID.String()
}

func MarshalNode(node Node) ([]byte, error) {
	return json.Marshal(node)
}

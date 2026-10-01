package placement

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"sort"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

const (
	nodePrefix      = "runtime:media:nodes:"
	drainingPrefix  = "runtime:media:draining:"
	leasesPrefix    = "runtime:media:leases:"
	ownershipPrefix = "runtime:media:sessions:"

	DefaultNodeTTL  = 15 * time.Second
	DefaultLeaseTTL = 45 * time.Second
)

var ErrNoCapacity = errors.New("no healthy media node has capacity")

type Node struct {
	ID         string `json:"id"`
	ControlURL string `json:"control_url"`
	Capacity   int    `json:"capacity"`
}

type Registry struct {
	redis    *redisintegration.Client
	nodeTTL  time.Duration
	leaseTTL time.Duration
}

func New(redis *redisintegration.Client) *Registry {
	if redis == nil {
		panic("media placement: Redis client is required")
	}
	return &Registry{
		redis:    redis,
		nodeTTL:  DefaultNodeTTL,
		leaseTTL: DefaultLeaseTTL,
	}
}

func (r *Registry) Heartbeat(ctx context.Context, node Node) error {
	if err := node.Validate(); err != nil {
		return err
	}
	return r.redis.SetJSON(ctx, nodeKey(node.ID), node, r.nodeTTL)
}

func (r *Registry) SetDraining(ctx context.Context, nodeID string, draining bool) error {
	nodeID = strings.TrimSpace(nodeID)
	if nodeID == "" {
		return fmt.Errorf("media node id is required")
	}
	if !draining {
		return r.redis.Delete(ctx, drainingKey(nodeID))
	}
	return r.redis.Set(ctx, drainingKey(nodeID), "1", r.nodeTTL)
}

func (r *Registry) Place(ctx context.Context, sessionID uuid.UUID) (Node, error) {
	if sessionID == uuid.Nil {
		return Node{}, fmt.Errorf("media session id is required")
	}
	nodes, err := r.nodes(ctx)
	if err != nil {
		return Node{}, err
	}

	type candidate struct {
		node Node
		free int64
	}
	candidates := make([]candidate, 0, len(nodes))
	for _, node := range nodes {
		draining, err := r.redis.Exists(ctx, drainingKey(node.ID))
		if err != nil {
			return Node{}, fmt.Errorf("read media node drain state: %w", err)
		}
		if draining {
			continue
		}
		active, err := r.redis.CountMediaLeases(ctx, leasesKey(node.ID))
		if err != nil {
			return Node{}, err
		}
		free := int64(node.Capacity) - active
		if free > 0 {
			candidates = append(candidates, candidate{node: node, free: free})
		}
	}
	sort.Slice(candidates, func(i, j int) bool {
		if candidates[i].free == candidates[j].free {
			return candidates[i].node.ID < candidates[j].node.ID
		}
		return candidates[i].free > candidates[j].free
	})

	for _, candidate := range candidates {
		ok, _, err := r.redis.AcquireMediaLease(
			ctx,
			nodeKey(candidate.node.ID),
			drainingKey(candidate.node.ID),
			leasesKey(candidate.node.ID),
			ownerKey(sessionID),
			candidate.node.ID,
			sessionID.String(),
			int64(candidate.node.Capacity),
			r.leaseTTL,
		)
		if err != nil {
			return Node{}, err
		}
		if ok {
			return candidate.node, nil
		}
	}
	return Node{}, ErrNoCapacity
}

func (r *Registry) Owner(ctx context.Context, sessionID uuid.UUID) (Node, error) {
	if sessionID == uuid.Nil {
		return Node{}, fmt.Errorf("media session id is required")
	}
	nodeID, err := r.redis.Get(ctx, ownerKey(sessionID))
	if errors.Is(err, redisv9.Nil) {
		return Node{}, ErrNoCapacity
	}
	if err != nil {
		return Node{}, fmt.Errorf("read media session owner: %w", err)
	}
	var node Node
	if err := r.redis.GetJSON(ctx, nodeKey(nodeID), &node); err != nil {
		return Node{}, fmt.Errorf("read media owner node: %w", err)
	}
	return node, nil
}

func (r *Registry) Refresh(ctx context.Context, nodeID string, sessionID uuid.UUID) error {
	return r.redis.RefreshMediaLease(
		ctx,
		nodeKey(nodeID),
		leasesKey(nodeID),
		ownerKey(sessionID),
		nodeID,
		sessionID.String(),
		r.leaseTTL,
	)
}

func (r *Registry) Release(ctx context.Context, nodeID string, sessionID uuid.UUID) error {
	if nodeID == "" || sessionID == uuid.Nil {
		return nil
	}
	return r.redis.ReleaseMediaLease(
		ctx,
		leasesKey(nodeID),
		ownerKey(sessionID),
		nodeID,
		sessionID.String(),
	)
}

func (r *Registry) nodes(ctx context.Context) ([]Node, error) {
	keys, err := r.redis.ScanKeys(ctx, nodePrefix+"*")
	if err != nil {
		return nil, err
	}
	nodes := make([]Node, 0, len(keys))
	for _, key := range keys {
		raw, err := r.redis.Get(ctx, key)
		if errors.Is(err, redisv9.Nil) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("read media node: %w", err)
		}
		var node Node
		if err := json.Unmarshal([]byte(raw), &node); err != nil {
			return nil, fmt.Errorf("decode media node: %w", err)
		}
		if err := node.Validate(); err != nil {
			continue
		}
		nodes = append(nodes, node)
	}
	return nodes, nil
}

func (n Node) Validate() error {
	if strings.TrimSpace(n.ID) == "" {
		return fmt.Errorf("media node id is required")
	}
	parsed, err := url.Parse(strings.TrimSpace(n.ControlURL))
	if err != nil || (parsed.Scheme != "http" && parsed.Scheme != "https") || parsed.Host == "" {
		return fmt.Errorf("media node control URL must be an absolute http or https URL")
	}
	if n.Capacity <= 0 {
		return fmt.Errorf("media node capacity must be positive")
	}
	return nil
}

func nodeKey(id string) string     { return nodePrefix + id }
func drainingKey(id string) string { return drainingPrefix + id }
func leasesKey(id string) string   { return leasesPrefix + id }
func ownerKey(id uuid.UUID) string { return ownershipPrefix + id.String() }

package redis

import (
	"context"
	"fmt"
	"time"
)

func (c *Client) ScanKeys(ctx context.Context, pattern string) ([]string, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}
	if ctx == nil || pattern == "" {
		return nil, fmt.Errorf("Redis scan context and pattern are required")
	}

	var (
		cursor uint64
		keys   []string
	)
	for {
		batch, next, err := c.client.Scan(ctx, cursor, pattern, 100).Result()
		if err != nil {
			return nil, fmt.Errorf("scan Redis keys: %w", err)
		}
		keys = append(keys, batch...)
		cursor = next
		if cursor == 0 {
			break
		}
	}
	return keys, nil
}

func (c *Client) AcquireMediaLease(
	ctx context.Context,
	nodeKey string,
	drainingKey string,
	leasesKey string,
	ownerKey string,
	nodeID string,
	sessionID string,
	capacity int64,
	ttl time.Duration,
) (bool, string, error) {
	if ctx == nil || nodeID == "" || sessionID == "" || capacity <= 0 || ttl <= 0 {
		return false, "", fmt.Errorf("valid media lease context, ids, capacity, and TTL are required")
	}

	const script = `
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 'missing'
end
if redis.call('EXISTS', KEYS[2]) == 1 then
  return 'draining'
end

local owner = redis.call('GET', KEYS[4])
if owner then
  if owner ~= ARGV[1] then
    return 'owned'
  end
  redis.call('ZADD', KEYS[3], ARGV[3], ARGV[2])
  redis.call('PEXPIRE', KEYS[3], ARGV[4])
  redis.call('PEXPIRE', KEYS[4], ARGV[4])
  return 'ok'
end

redis.call('ZREMRANGEBYSCORE', KEYS[3], '-inf', ARGV[5])
if redis.call('ZCARD', KEYS[3]) >= tonumber(ARGV[6]) then
  return 'capacity'
end

redis.call('ZADD', KEYS[3], ARGV[3], ARGV[2])
redis.call('PEXPIRE', KEYS[3], ARGV[4])
redis.call('SET', KEYS[4], ARGV[1], 'PX', ARGV[4])
return 'ok'
`

	now := time.Now()
	expiry := now.Add(ttl)
	reason, err := c.client.Eval(
		ctx,
		script,
		[]string{nodeKey, drainingKey, leasesKey, ownerKey},
		nodeID,
		sessionID,
		expiry.UnixMilli(),
		ttl.Milliseconds(),
		now.UnixMilli(),
		capacity,
	).Text()
	if err != nil {
		return false, "", fmt.Errorf("acquire Redis media lease: %w", err)
	}
	return reason == "ok", reason, nil
}

func (c *Client) RefreshMediaLease(
	ctx context.Context,
	nodeKey string,
	leasesKey string,
	ownerKey string,
	nodeID string,
	sessionID string,
	ttl time.Duration,
) error {
	if ctx == nil || nodeID == "" || sessionID == "" || ttl <= 0 {
		return fmt.Errorf("valid media lease refresh context, ids, and TTL are required")
	}

	const script = `
if redis.call('EXISTS', KEYS[1]) == 0 then
  return 0
end
local owner = redis.call('GET', KEYS[3])
if not owner or owner ~= ARGV[1] then
  return 0
end
redis.call('ZADD', KEYS[2], ARGV[3], ARGV[2])
redis.call('PEXPIRE', KEYS[2], ARGV[4])
redis.call('PEXPIRE', KEYS[3], ARGV[4])
return 1
`
	ok, err := c.client.Eval(
		ctx,
		script,
		[]string{nodeKey, leasesKey, ownerKey},
		nodeID,
		sessionID,
		time.Now().Add(ttl).UnixMilli(),
		ttl.Milliseconds(),
	).Bool()
	if err != nil {
		return fmt.Errorf("refresh Redis media lease: %w", err)
	}
	if !ok {
		return fmt.Errorf("media session ownership is unavailable")
	}
	return nil
}

func (c *Client) ReleaseMediaLease(
	ctx context.Context,
	leasesKey string,
	ownerKey string,
	nodeID string,
	sessionID string,
) error {
	if ctx == nil || nodeID == "" || sessionID == "" {
		return fmt.Errorf("valid media lease release context and ids are required")
	}

	const script = `
redis.call('ZREM', KEYS[1], ARGV[2])
local owner = redis.call('GET', KEYS[2])
if owner and owner == ARGV[1] then
  redis.call('DEL', KEYS[2])
end
return 1
`
	if err := c.client.Eval(
		ctx,
		script,
		[]string{leasesKey, ownerKey},
		nodeID,
		sessionID,
	).Err(); err != nil {
		return fmt.Errorf("release Redis media lease: %w", err)
	}
	return nil
}

func (c *Client) CountMediaLeases(
	ctx context.Context,
	leasesKey string,
) (int64, error) {
	if ctx == nil || leasesKey == "" {
		return 0, fmt.Errorf("media lease count context and key are required")
	}
	now := time.Now().UnixMilli()
	if err := c.client.ZRemRangeByScore(ctx, leasesKey, "-inf", fmt.Sprintf("%d", now)).Err(); err != nil {
		return 0, fmt.Errorf("prune Redis media leases: %w", err)
	}
	count, err := c.client.ZCard(ctx, leasesKey).Result()
	if err != nil {
		return 0, fmt.Errorf("count Redis media leases: %w", err)
	}
	return count, nil
}

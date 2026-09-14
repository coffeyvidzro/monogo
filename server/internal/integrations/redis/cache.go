package redis

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

// Get returns the value stored at key. A missing key returns redis.Nil.
func (c *Client) Get(ctx context.Context, key string) (string, error) {
	if err := c.validateKey(key); err != nil {
		return "", err
	}
	if ctx == nil {
		return "", fmt.Errorf("redis context is nil")
	}

	return c.client.Get(ctx, key).Result()
}

// Set stores value at key. A zero TTL stores the value without expiry.
func (c *Client) Set(ctx context.Context, key string, value any, ttl time.Duration) error {
	if err := c.validateKey(key); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("redis context is nil")
	}

	return c.client.Set(ctx, key, value, ttl).Err()
}

// Delete removes keys. It is a no-op when no keys are provided.
func (c *Client) Delete(ctx context.Context, keys ...string) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("redis context is nil")
	}
	if len(keys) == 0 {
		return nil
	}

	for _, key := range keys {
		if key == "" {
			return fmt.Errorf("redis key is required")
		}
	}

	return c.client.Del(ctx, keys...).Err()
}

// Exists reports whether key exists.
func (c *Client) Exists(ctx context.Context, key string) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}
	if ctx == nil {
		return false, fmt.Errorf("redis context is nil")
	}

	count, err := c.client.Exists(ctx, key).Result()
	if err != nil {
		return false, err
	}

	return count > 0, nil
}

// AllowFixedWindow atomically increments a bounded, expiring counter and
// reports whether the request remains within limit. The Lua script ensures
// every API replica shares the same decision and that counters cannot persist
// without an expiry after a process failure.
func (c *Client) AllowFixedWindow(ctx context.Context, key string, limit int64, window time.Duration) (bool, error) {
	if err := c.validateKey(key); err != nil {
		return false, err
	}
	if ctx == nil {
		return false, fmt.Errorf("redis context is nil")
	}
	if limit <= 0 {
		return false, fmt.Errorf("rate limit must be positive")
	}
	if window <= 0 {
		return false, fmt.Errorf("rate limit window must be positive")
	}

	const script = `
local current = redis.call('INCR', KEYS[1])
if current == 1 then
  redis.call('PEXPIRE', KEYS[1], ARGV[1])
end
if current <= tonumber(ARGV[2]) then
  return 1
end
return 0
`
	allowed, err := c.client.Eval(ctx, script, []string{key}, window.Milliseconds(), limit).Bool()
	if err != nil {
		return false, fmt.Errorf("apply Redis fixed-window rate limit: %w", err)
	}
	return allowed, nil
}

// AcquireCallLease atomically applies a per-second admission counter and a
// concurrent-call lease. Expired leases are removed before counting so a
// crashed process cannot hold capacity forever.
func (c *Client) AcquireCallLease(ctx context.Context, prefix, leaseID string, maxCPS, maxConcurrent int64, ttl time.Duration) (bool, string, error) {
	if err := c.validateKey(prefix); err != nil {
		return false, "", err
	}
	if ctx == nil || leaseID == "" || maxCPS <= 0 || maxConcurrent <= 0 || ttl <= 0 {
		return false, "", fmt.Errorf("valid call lease context, id, limits, and TTL are required")
	}
	const script = `
redis.call('ZREMRANGEBYSCORE', KEYS[2], '-inf', ARGV[1])
local cps = redis.call('INCR', KEYS[1])
if cps == 1 then redis.call('PEXPIRE', KEYS[1], 1000) end
if cps > tonumber(ARGV[2]) then return 'cps' end
if redis.call('ZCARD', KEYS[2]) >= tonumber(ARGV[3]) then return 'concurrent' end
redis.call('ZADD', KEYS[2], ARGV[4], ARGV[5])
redis.call('PEXPIRE', KEYS[2], ARGV[6])
return 'ok'
`
	now := time.Now().UnixMilli()
	reason, err := c.client.Eval(ctx, script, []string{prefix + ":cps", prefix + ":leases"}, now, maxCPS, maxConcurrent, now+ttl.Milliseconds(), leaseID, ttl.Milliseconds()).Text()
	if err != nil {
		return false, "", fmt.Errorf("acquire Redis call lease: %w", err)
	}
	return reason == "ok", reason, nil
}

func (c *Client) BindCallLease(ctx context.Context, prefix, leaseID, callID string) error {
	if err := c.validateKey(prefix); err != nil {
		return err
	}
	if ctx == nil || leaseID == "" || callID == "" {
		return fmt.Errorf("call lease context, lease id, and call id are required")
	}
	const script = `
local score = redis.call('ZSCORE', KEYS[1], ARGV[1])
if not score then return 0 end
redis.call('ZREM', KEYS[1], ARGV[1])
redis.call('ZADD', KEYS[1], score, ARGV[2])
return 1
`
	bound, err := c.client.Eval(ctx, script, []string{prefix + ":leases"}, leaseID, callID).Bool()
	if err != nil {
		return fmt.Errorf("bind Redis call lease: %w", err)
	}
	if !bound {
		return fmt.Errorf("call lease expired before binding")
	}
	return nil
}

func (c *Client) ReleaseCallLease(ctx context.Context, prefix, callOrLeaseID string) error {
	if err := c.validateKey(prefix); err != nil {
		return err
	}
	if ctx == nil || callOrLeaseID == "" {
		return fmt.Errorf("call lease context and id are required")
	}
	if err := c.client.ZRem(ctx, prefix+":leases", callOrLeaseID).Err(); err != nil {
		return fmt.Errorf("release Redis call lease: %w", err)
	}
	return nil
}

// RefreshCallLease renews or reconstructs a lease from durable active-call
// state. Reconciliation uses this after worker or Redis restarts.
func (c *Client) RefreshCallLease(ctx context.Context, prefix, callID string, ttl time.Duration) error {
	if err := c.validateKey(prefix); err != nil {
		return err
	}
	if ctx == nil || callID == "" || ttl <= 0 {
		return fmt.Errorf("call lease context, id, and TTL are required")
	}
	key := prefix + ":leases"
	if err := c.client.ZAdd(ctx, key, redisv9.Z{Score: float64(time.Now().Add(ttl).UnixMilli()), Member: callID}).Err(); err != nil {
		return fmt.Errorf("refresh Redis call lease: %w", err)
	}
	if err := c.client.PExpire(ctx, key, ttl).Err(); err != nil {
		return fmt.Errorf("expire refreshed Redis call leases: %w", err)
	}
	return nil
}

// IncrementMetric updates a shared bounded metric series. New series are
// refused once maxSeries is reached, preventing tenant-created resources from
// causing unbounded label cardinality.
func (c *Client) IncrementMetric(ctx context.Context, field string, maxSeries int64) error {
	if ctx == nil || field == "" || maxSeries <= 0 {
		return fmt.Errorf("metric context, field, and series limit are required")
	}
	const script = `
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 and redis.call('HLEN', KEYS[1]) >= tonumber(ARGV[2]) then
  return 0
end
redis.call('HINCRBY', KEYS[1], ARGV[1], 1)
return 1
`
	if _, err := c.client.Eval(ctx, script, []string{"telecom:metrics:counters"}, field, maxSeries).Result(); err != nil {
		return fmt.Errorf("increment shared telecom metric: %w", err)
	}
	return nil
}

func (c *Client) SetMetricGauge(ctx context.Context, field string, value float64, maxSeries int64) error {
	if ctx == nil || field == "" || maxSeries <= 0 {
		return fmt.Errorf("metric context, field, and series limit are required")
	}
	const script = `
if redis.call('HEXISTS', KEYS[1], ARGV[1]) == 0 and redis.call('HLEN', KEYS[1]) >= tonumber(ARGV[3]) then
  return 0
end
redis.call('HSET', KEYS[1], ARGV[1], ARGV[2])
return 1
`
	if _, err := c.client.Eval(ctx, script, []string{"telecom:metrics:gauges"}, field, value, maxSeries).Result(); err != nil {
		return fmt.Errorf("set shared telecom metric gauge: %w", err)
	}
	return nil
}

func (c *Client) TelecomMetrics(ctx context.Context) (map[string]string, map[string]string, error) {
	if ctx == nil {
		return nil, nil, fmt.Errorf("metric context is required")
	}
	counters, err := c.client.HGetAll(ctx, "telecom:metrics:counters").Result()
	if err != nil {
		return nil, nil, fmt.Errorf("read shared telecom counters: %w", err)
	}
	gauges, err := c.client.HGetAll(ctx, "telecom:metrics:gauges").Result()
	if err != nil {
		return nil, nil, fmt.Errorf("read shared telecom gauges: %w", err)
	}
	return counters, gauges, nil
}

// GetJSON retrieves a JSON value into dst.
func (c *Client) GetJSON(ctx context.Context, key string, dst any) error {
	if dst == nil {
		return fmt.Errorf("redis destination is nil")
	}

	value, err := c.Get(ctx, key)
	if err != nil {
		return err
	}

	if err := json.Unmarshal([]byte(value), dst); err != nil {
		return fmt.Errorf("decode redis value: %w", err)
	}

	return nil
}

// SetJSON JSON-encodes value and stores it at key.
func (c *Client) SetJSON(ctx context.Context, key string, value any, ttl time.Duration) error {
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Errorf("encode redis value: %w", err)
	}

	return c.Set(ctx, key, encoded, ttl)
}

func (c *Client) validateKey(key string) error {
	if err := c.validate(); err != nil {
		return err
	}
	if key == "" {
		return fmt.Errorf("redis key is required")
	}

	return nil
}

package redis

import (
	"context"
	"fmt"
	"strings"
	"time"

	redisv9 "github.com/redis/go-redis/v9"
)

type Config struct {
	URL             string
	MaxRetries      int
	MinIdleConns    int
	MaxIdleConns    int
	ConnMaxIdleTime time.Duration
	ConnMaxLifetime time.Duration
	DialTimeout     time.Duration
	ReadTimeout     time.Duration
	WriteTimeout    time.Duration
}

func DefaultConfig(redisURL string) Config {
	return Config{
		URL:             strings.TrimSpace(redisURL),
		MaxRetries:      3,
		MinIdleConns:    2,
		MaxIdleConns:    20,
		ConnMaxIdleTime: 30 * time.Minute,
		ConnMaxLifetime: time.Hour,
		DialTimeout:     5 * time.Second,
		ReadTimeout:     3 * time.Second,
		WriteTimeout:    3 * time.Second,
	}
}

type Client struct {
	client *redisv9.Client
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("redis URL is required")
	}
	if ctx == nil {
		return nil, fmt.Errorf("redis context is nil")
	}

	options, err := redisv9.ParseURL(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse redis URL: %w", err)
	}

	options.MaxRetries = cfg.MaxRetries
	options.MinIdleConns = cfg.MinIdleConns
	options.MaxIdleConns = cfg.MaxIdleConns
	options.ConnMaxIdleTime = cfg.ConnMaxIdleTime
	options.ConnMaxLifetime = cfg.ConnMaxLifetime
	options.DialTimeout = cfg.DialTimeout
	options.ReadTimeout = cfg.ReadTimeout
	options.WriteTimeout = cfg.WriteTimeout

	client := redisv9.NewClient(options)
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("ping redis: %w", err)
	}

	return &Client{client: client}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("redis context is nil")
	}

	return c.client.Ping(ctx).Err()
}

func (c *Client) Close() error {
	if c == nil || c.client == nil {
		return nil
	}

	return c.client.Close()
}

func (c *Client) validate() error {
	if c == nil || c.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	return nil
}

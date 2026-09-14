package postgres

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// Config configures the PostgreSQL connection pool.
type Config struct {
	URL               string
	MaxConnections    int32
	MinConnections    int32
	MaxLifetime       time.Duration
	MaxIdleTime       time.Duration
	HealthCheckPeriod time.Duration
}

// DefaultConfig returns the default PostgreSQL connection pool configuration.
func DefaultConfig(databaseURL string) Config {
	return Config{
		URL:               strings.TrimSpace(databaseURL),
		MaxConnections:    20,
		MinConnections:    2,
		MaxLifetime:       time.Hour,
		MaxIdleTime:       30 * time.Minute,
		HealthCheckPeriod: time.Minute,
	}
}

// Client owns the PostgreSQL connection pool used by the application.
type Client struct {
	pool *pgxpool.Pool
}

// New creates a PostgreSQL client and establishes the connection pool.
func New(ctx context.Context, cfg Config) (*Client, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("postgres URL is required")
	}

	poolConfig, err := pgxpool.ParseConfig(cfg.URL)
	if err != nil {
		return nil, fmt.Errorf("parse postgres config: %w", err)
	}

	if cfg.MaxConnections > 0 {
		poolConfig.MaxConns = cfg.MaxConnections
	}
	if cfg.MinConnections > 0 {
		poolConfig.MinConns = cfg.MinConnections
	}
	if cfg.MaxLifetime > 0 {
		poolConfig.MaxConnLifetime = cfg.MaxLifetime
	}
	if cfg.MaxIdleTime > 0 {
		poolConfig.MaxConnIdleTime = cfg.MaxIdleTime
	}
	if cfg.HealthCheckPeriod > 0 {
		poolConfig.HealthCheckPeriod = cfg.HealthCheckPeriod
	}

	pool, err := pgxpool.NewWithConfig(ctx, poolConfig)
	if err != nil {
		return nil, fmt.Errorf("create postgres pool: %w", err)
	}

	if err := pool.Ping(ctx); err != nil {
		pool.Close()
		return nil, fmt.Errorf("ping postgres: %w", err)
	}

	return &Client{pool: pool}, nil
}

// Pool returns the underlying PostgreSQL connection pool.
func (c *Client) Pool() *pgxpool.Pool {
	return c.pool
}

// Ping verifies that PostgreSQL is reachable.
func (c *Client) Ping(ctx context.Context) error {
	return c.pool.Ping(ctx)
}

// Close releases all connections owned by the client.
func (c *Client) Close() {
	if c.pool != nil {
		c.pool.Close()
	}
}

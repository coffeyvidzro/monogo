package nats

import (
	"context"
	"fmt"
	"strings"
	"time"

	natsgo "github.com/nats-io/nats.go"
	natsjs "github.com/nats-io/nats.go/jetstream"
)

const (
	defaultClientName     = "leamout"
	defaultConnectTimeout = 5 * time.Second
	defaultReconnectWait  = 2 * time.Second
	defaultDrainTimeout   = 10 * time.Second
)

type Config struct {
	URL            string
	ClientName     string
	ConnectTimeout time.Duration
	ReconnectWait  time.Duration
	MaxReconnects  int
	DrainTimeout   time.Duration
}

func DefaultConfig(natsURL string) Config {
	return Config{
		URL:            strings.TrimSpace(natsURL),
		ClientName:     defaultClientName,
		ConnectTimeout: defaultConnectTimeout,
		ReconnectWait:  defaultReconnectWait,
		MaxReconnects:  -1,
		DrainTimeout:   defaultDrainTimeout,
	}
}

type Client struct {
	connection   *natsgo.Conn
	jetStream    natsjs.JetStream
	drainTimeout time.Duration
}

func New(ctx context.Context, cfg Config) (*Client, error) {
	if ctx == nil {
		return nil, fmt.Errorf("NATS context is required")
	}
	if strings.TrimSpace(cfg.URL) == "" {
		return nil, fmt.Errorf("NATS URL is required")
	}
	if strings.TrimSpace(cfg.ClientName) == "" {
		cfg.ClientName = defaultClientName
	}
	if cfg.ConnectTimeout <= 0 {
		cfg.ConnectTimeout = defaultConnectTimeout
	}
	if cfg.ReconnectWait <= 0 {
		cfg.ReconnectWait = defaultReconnectWait
	}
	if cfg.DrainTimeout <= 0 {
		cfg.DrainTimeout = defaultDrainTimeout
	}

	conn, err := natsgo.Connect(
		cfg.URL,
		natsgo.Name(cfg.ClientName),
		natsgo.Timeout(cfg.ConnectTimeout),
		natsgo.MaxReconnects(cfg.MaxReconnects),
		natsgo.ReconnectWait(cfg.ReconnectWait),
	)
	if err != nil {
		return nil, fmt.Errorf("connect to NATS: %w", err)
	}

	js, err := natsjs.New(conn)
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("initialize JetStream: %w", err)
	}

	if _, err := js.AccountInfo(ctx); err != nil {
		conn.Close()
		return nil, fmt.Errorf("verify JetStream: %w", err)
	}

	return &Client{connection: conn, jetStream: js, drainTimeout: cfg.DrainTimeout}, nil
}

func (c *Client) Ping(ctx context.Context) error {
	if err := c.validate(); err != nil {
		return err
	}
	if ctx == nil {
		return fmt.Errorf("NATS context is required")
	}
	if !c.connection.IsConnected() {
		return fmt.Errorf("NATS connection is unavailable")
	}

	return c.connection.FlushWithContext(ctx)
}

func (c *Client) Close() error {
	if c == nil || c.connection == nil || c.connection.IsClosed() {
		return nil
	}

	done := make(chan error, 1)
	go func() { done <- c.connection.Drain() }()

	ctx, cancel := context.WithTimeout(context.Background(), c.drainTimeout)
	defer cancel()

	select {
	case err := <-done:
		if err != nil {
			c.connection.Close()
			return fmt.Errorf("drain NATS connection: %w", err)
		}
	case <-ctx.Done():
		c.connection.Close()
		return fmt.Errorf("drain NATS connection: %w", ctx.Err())
	}

	return nil
}

func (c *Client) JetStream() (natsjs.JetStream, error) {
	if err := c.validate(); err != nil {
		return nil, err
	}

	return c.jetStream, nil
}

func (c *Client) validate() error {
	if c == nil || c.connection == nil || c.jetStream == nil {
		return fmt.Errorf("NATS client is nil")
	}

	return nil
}

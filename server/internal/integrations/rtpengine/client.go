package rtpengine

import (
	"context"
	"errors"
	"fmt"
	"net"
	"time"
)

// MediaProxy defines the contract for RTPEngine operations.
type MediaProxy interface {
	HealthCheck(ctx context.Context) error
	Close() error
	Offer(ctx context.Context, request OfferRequest) (OfferResponse, error)
	Answer(ctx context.Context, request AnswerRequest) (AnswerResponse, error)
	Delete(ctx context.Context, request DeleteRequest) error
	Query(ctx context.Context, session Session) (QueryResponse, error)
}

// Client implements MediaProxy using RTPEngine's UDP control protocol.
type Client struct {
	address  string
	timeout  time.Duration
	dialer   net.Dialer
	maxRetry int
}

func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		address:  cfg.Address,
		timeout:  cfg.CommandTimeout,
		dialer:   net.Dialer{Timeout: cfg.ConnectTimeout},
		maxRetry: cfg.MaxRetries,
	}, nil
}

// HealthCheck verifies that RTPEngine responds to a control command.
func (c *Client) HealthCheck(ctx context.Context) error {
	if ctx == nil {
		return errors.New("RTPEngine context is required")
	}

	if _, err := c.do(ctx, CommandPing, nil); err != nil {
		return fmt.Errorf("RTPEngine health check failed: %w", err)
	}
	return nil
}

// Close is a no-op because each command owns its UDP connection.
func (c *Client) Close() error {
	return nil
}

func (c *Client) do(ctx context.Context, command Command, params map[string]any) (Response, error) {
	if ctx == nil {
		return Response{}, errors.New("RTPEngine context is required")
	}
	if command == "" {
		return Response{}, errors.New("RTPEngine command is required")
	}

	var lastErr error
	for attempt := 0; attempt <= c.maxRetry; attempt++ {
		response, err := c.doOnce(ctx, command, params)
		if err == nil {
			return response, nil
		}
		lastErr = err

		if ctx.Err() != nil {
			return Response{}, ctx.Err()
		}
		if !retryable(err) || attempt == c.maxRetry {
			break
		}

		if err := waitRetry(ctx, 50*time.Millisecond); err != nil {
			return Response{}, err
		}
	}

	return Response{}, fmt.Errorf(
		"RTPEngine command failed after %d attempts: %w",
		c.maxRetry+1,
		lastErr,
	)
}

func (c *Client) doOnce(ctx context.Context, command Command, params map[string]any) (Response, error) {
	conn, err := c.dialer.DialContext(ctx, "udp", c.address)
	if err != nil {
		return Response{}, fmt.Errorf("dial RTPEngine: %w", err)
	}
	defer func() { _ = conn.Close() }()

	deadline := time.Now().Add(c.timeout)
	if contextDeadline, ok := ctx.Deadline(); ok && contextDeadline.Before(deadline) {
		deadline = contextDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		return Response{}, fmt.Errorf("set RTPEngine deadline: %w", err)
	}

	packet, cookie, err := encodeRequest(command, params)
	if err != nil {
		return Response{}, err
	}
	if _, err := conn.Write(packet); err != nil {
		return Response{}, fmt.Errorf("write RTPEngine request: %w", err)
	}

	buffer := make([]byte, 1<<20)
	n, err := conn.Read(buffer)
	if err != nil {
		return Response{}, fmt.Errorf("read RTPEngine response: %w", err)
	}

	return decodeResponse(buffer[:n], cookie)
}

func retryable(err error) bool {
	var netErr net.Error
	return errors.As(err, &netErr) && netErr.Timeout()
}

func waitRetry(ctx context.Context, delay time.Duration) error {
	timer := time.NewTimer(delay)
	defer timer.Stop()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

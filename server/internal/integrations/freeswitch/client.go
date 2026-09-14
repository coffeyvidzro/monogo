package freeswitch

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net"
	"strconv"
	"strings"
	"sync"
	"time"
)

// MediaController defines the contract for FreeSWITCH media server operations.
// This interface enables testability and allows swapping implementations.
type MediaController interface {
	Connect(ctx context.Context) error
	Close() error
	HealthCheck(ctx context.Context) error
	Command(ctx context.Context, command string) (Reply, error)
	BGAPI(ctx context.Context, command string) (Job, error)
	Subscribe(ctx context.Context, format EventFormat, events []string, handler EventHandler) error
}

type Client struct {
	address           string
	password          string
	connectTimeout    time.Duration
	commandTimeout    time.Duration
	reconnectMinDelay time.Duration
	reconnectMaxDelay time.Duration
	dialer            *net.Dialer

	mu           sync.Mutex
	conn         net.Conn
	reader       *bufio.Reader
	done         chan struct{}
	readErr      chan error
	replyCh      chan Frame
	ready        bool
	closed       bool
	reconnecting bool

	writeMu   sync.Mutex
	commandMu sync.Mutex

	subscriptionsMu sync.RWMutex
	subscriptions   []subscription

	lifecycleCtx    context.Context
	lifecycleCancel context.CancelFunc
	wg              sync.WaitGroup
}

func New(cfg Config) (*Client, error) {
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	lifecycleCtx, lifecycleCancel := context.WithCancel(context.Background())

	return &Client{
		address:           cfg.Address,
		password:          cfg.Password,
		connectTimeout:    cfg.ConnectTimeout,
		commandTimeout:    cfg.CommandTimeout,
		reconnectMinDelay: cfg.ReconnectMinDelay,
		reconnectMaxDelay: cfg.ReconnectMaxDelay,
		dialer:            &net.Dialer{Timeout: cfg.ConnectTimeout},
		lifecycleCtx:      lifecycleCtx,
		lifecycleCancel:   lifecycleCancel,
	}, nil
}

func (c *Client) Connect(ctx context.Context) error {
	if ctx == nil {
		return fmt.Errorf("FreeSWITCH context is required")
	}

	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return fmt.Errorf("FreeSWITCH client is closed")
	}
	if c.conn != nil {
		c.mu.Unlock()
		return nil
	}
	c.mu.Unlock()

	if err := c.connectAndRestore(ctx); err != nil {
		return err
	}
	return nil
}

func (c *Client) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return nil
	}
	c.closed = true
	c.ready = false
	conn := c.conn
	done := c.done
	c.mu.Unlock()

	c.lifecycleCancel()

	var closeErr error
	if conn != nil {
		closeErr = conn.Close()
	}
	if done != nil {
		<-done
	}

	c.wg.Wait()
	return closeErr
}

func (c *Client) Ready() bool {
	c.mu.Lock()
	defer c.mu.Unlock()

	return !c.closed && c.conn != nil && c.ready
}

func (c *Client) HealthCheck(ctx context.Context) error {
	if !c.Ready() {
		return fmt.Errorf("FreeSWITCH client is not ready")
	}

	ctx, cancel := context.WithTimeout(ctx, 2*time.Second)
	defer cancel()

	reply, err := c.Command(ctx, "status")
	if err != nil {
		return fmt.Errorf("FreeSWITCH health check failed: %w", err)
	}
	if !freeSWITCHStatusUp(reply) {
		return fmt.Errorf("FreeSWITCH is not running properly")
	}
	return nil
}

func freeSWITCHStatusUp(reply Reply) bool {
	for _, value := range []string{reply.Body, reply.Text} {
		status := strings.ToUpper(strings.TrimSpace(value))
		if status == "UP" ||
			strings.HasPrefix(status, "UP ") ||
			strings.HasPrefix(status, "+OK UP") {
			return true
		}
	}

	return false
}

func (c *Client) command(ctx context.Context, command string) (Frame, error) {
	if ctx == nil {
		return Frame{}, fmt.Errorf("FreeSWITCH context is required")
	}
	command = strings.TrimSpace(command)
	if command == "" {
		return Frame{}, fmt.Errorf("FreeSWITCH command is required")
	}

	c.commandMu.Lock()
	defer c.commandMu.Unlock()
	commandCtx, cancel := context.WithTimeout(ctx, c.commandTimeout)
	defer cancel()

	c.mu.Lock()
	conn := c.conn
	replyCh := c.replyCh
	done := c.done
	readErr := c.readErr
	c.mu.Unlock()
	if conn == nil || replyCh == nil || done == nil || readErr == nil {
		return Frame{}, fmt.Errorf("FreeSWITCH client is not connected")
	}

	deadline, _ := commandCtx.Deadline()
	if err := conn.SetWriteDeadline(deadline); err != nil {
		return Frame{}, fmt.Errorf("set FreeSWITCH command deadline: %w", err)
	}

	c.writeMu.Lock()
	_, err := writeCommand(conn, command)
	clearDeadlineErr := conn.SetWriteDeadline(time.Time{})
	c.writeMu.Unlock()
	if err != nil {
		return Frame{}, fmt.Errorf("write FreeSWITCH command: %w", err)
	}
	if clearDeadlineErr != nil {
		return Frame{}, fmt.Errorf("clear FreeSWITCH command deadline: %w", clearDeadlineErr)
	}

	select {
	case frame := <-replyCh:
		return frame, nil
	case err := <-readErr:
		return Frame{}, fmt.Errorf("read FreeSWITCH response: %w", err)
	case <-done:
		return Frame{}, fmt.Errorf("FreeSWITCH connection closed")
	case <-commandCtx.Done():
		// A timed-out response can arrive during a later command and corrupt the
		// request/response pairing. Retire this connection and let the supervisor
		// establish a clean ESL session before accepting another command.
		c.disconnect(conn, commandCtx.Err())
		return Frame{}, fmt.Errorf("FreeSWITCH command: %w", commandCtx.Err())
	}
}

func (c *Client) readLoop(
	conn net.Conn,
	reader *bufio.Reader,
	replyCh chan<- Frame,
	done <-chan struct{},
) {
	defer c.wg.Done()

	for {
		frame, err := readFrame(reader)
		if err != nil {
			c.disconnect(conn, err)
			return
		}

		switch frame.ContentType {
		case ContentTypeCommandReply, ContentTypeAPIResponse:
			select {
			case replyCh <- frame:
			case <-done:
				return
			case <-c.lifecycleCtx.Done():
				c.disconnect(conn, context.Canceled)
				return
			}
		case ContentTypeEventPlain:
			c.dispatchEvent(frame)
		}
	}
}

func (c *Client) disconnect(conn net.Conn, readErr error) {
	c.mu.Lock()
	if c.conn != conn {
		c.mu.Unlock()
		return
	}

	done := c.done
	errCh := c.readErr
	c.conn = nil
	c.reader = nil
	c.done = nil
	c.readErr = nil
	c.replyCh = nil
	c.ready = false
	shouldReconnect := !c.closed
	c.mu.Unlock()
	_ = conn.Close()

	if readErr == nil {
		readErr = io.EOF
	}
	if errCh != nil {
		select {
		case errCh <- readErr:
		default:
		}
	}
	if done != nil {
		close(done)
	}

	if shouldReconnect {
		c.startReconnect()
	}
}

func (c *Client) dispatchEvent(frame Frame) {
	event := Event{
		Headers: frame.Headers,
		Body:    frame.Body,
		Name:    frame.Header("Event-Name"),
	}

	for _, subscription := range c.subscriptionSnapshot() {
		if !subscription.matches(event.Name) {
			continue
		}

		go func(handler EventHandler) {
			ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
			defer cancel()
			_ = handler(ctx, event)
		}(subscription.handler)
	}
}

func writeCommand(conn net.Conn, command string) (int, error) {
	command = strings.TrimSpace(command)
	if strings.ContainsAny(command, "\r\n") {
		return 0, fmt.Errorf("FreeSWITCH command cannot contain line breaks")
	}
	return fmt.Fprintf(conn, "%s\n\n", command)
}

func readFrame(reader *bufio.Reader) (Frame, error) {
	headers := make(map[string]string)
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			return Frame{}, err
		}
		line = strings.TrimRight(line, "\r\n")
		if line == "" {
			break
		}
		key, value, ok := strings.Cut(line, ": ")
		if !ok {
			return Frame{}, fmt.Errorf("invalid FreeSWITCH header %q", line)
		}
		headers[key] = value
	}

	length := 0
	if raw := headers["Content-Length"]; raw != "" {
		var err error
		length, err = strconv.Atoi(raw)
		if err != nil || length < 0 {
			return Frame{}, fmt.Errorf("invalid FreeSWITCH content length %q", raw)
		}
	}

	body := make([]byte, length)
	if length > 0 {
		if _, err := io.ReadFull(reader, body); err != nil {
			return Frame{}, fmt.Errorf("read FreeSWITCH frame body: %w", err)
		}
	}

	return Frame{
		ContentType: headers["Content-Type"],
		Headers:     headers,
		Body:        string(body),
	}, nil
}

package freeswitch

import (
	"bufio"
	"context"
	"fmt"
	"net"
	"time"
)

func (c *Client) connectAndRestore(ctx context.Context) error {
	conn, reader, err := c.connectAuthenticated(ctx)
	if err != nil {
		return err
	}

	installed, err := c.installConnection(conn, reader)
	if err != nil {
		_ = conn.Close()
		return err
	}
	if !installed {
		_ = conn.Close()
		return nil
	}

	if err := c.restoreSubscriptions(ctx); err != nil {
		_ = conn.Close()
		return fmt.Errorf("restore FreeSWITCH subscriptions: %w", err)
	}

	c.mu.Lock()
	if c.conn == conn && !c.closed {
		c.ready = true
	}
	c.mu.Unlock()

	return nil
}

func (c *Client) connectAuthenticated(ctx context.Context) (net.Conn, *bufio.Reader, error) {
	conn, err := c.dialer.DialContext(ctx, "tcp", c.address)
	if err != nil {
		return nil, nil, fmt.Errorf("connect to FreeSWITCH: %w", err)
	}

	stopCancel := context.AfterFunc(ctx, func() {
		_ = conn.Close()
	})
	defer stopCancel()

	deadline := time.Now().Add(c.connectTimeout)
	if ctxDeadline, ok := ctx.Deadline(); ok && ctxDeadline.Before(deadline) {
		deadline = ctxDeadline
	}
	if err := conn.SetDeadline(deadline); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("set FreeSWITCH authentication deadline: %w", err)
	}

	reader := bufio.NewReader(conn)
	frame, err := readFrame(reader)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("read FreeSWITCH authentication request: %w", err)
	}
	if frame.ContentType != ContentTypeAuthRequest {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("unexpected FreeSWITCH greeting: %q", frame.ContentType)
	}

	if _, err := writeCommand(conn, "auth "+c.password); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("write FreeSWITCH authentication command: %w", err)
	}

	frame, err = readFrame(reader)
	if err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("read FreeSWITCH authentication response: %w", err)
	}
	if frame.ContentType != ContentTypeCommandReply || !frame.OK() {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("FreeSWITCH authentication failed: %s", frame.ReplyText())
	}

	if err := conn.SetDeadline(time.Time{}); err != nil {
		_ = conn.Close()
		return nil, nil, fmt.Errorf("clear FreeSWITCH authentication deadline: %w", err)
	}

	return conn, reader, nil
}

func (c *Client) installConnection(conn net.Conn, reader *bufio.Reader) (bool, error) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return false, fmt.Errorf("FreeSWITCH client is closed")
	}
	if c.conn != nil {
		return false, nil
	}

	replyCh := make(chan Frame)
	done := make(chan struct{})
	c.conn = conn
	c.reader = reader
	c.done = done
	c.readErr = make(chan error, 1)
	c.replyCh = replyCh
	c.ready = false

	c.wg.Add(1)
	go c.readLoop(conn, reader, replyCh, done)

	return true, nil
}

func (c *Client) startReconnect() {
	c.mu.Lock()
	if c.closed || c.reconnecting {
		c.mu.Unlock()
		return
	}
	c.reconnecting = true
	c.wg.Add(1)
	c.mu.Unlock()

	go c.reconnectLoop()
}

func (c *Client) reconnectLoop() {
	defer c.wg.Done()
	defer func() {
		c.mu.Lock()
		c.reconnecting = false
		c.mu.Unlock()
	}()

	for attempt := 0; ; attempt++ {
		if err := c.connectAndRestore(c.lifecycleCtx); err == nil {
			return
		}

		delay := c.reconnectDelay(attempt)
		timer := time.NewTimer(delay)
		select {
		case <-c.lifecycleCtx.Done():
			if !timer.Stop() {
				<-timer.C
			}
			return
		case <-timer.C:
		}
	}
}

func (c *Client) reconnectDelay(attempt int) time.Duration {
	delay := c.reconnectMinDelay
	for i := 0; i < attempt && delay < c.reconnectMaxDelay; i++ {
		if delay > c.reconnectMaxDelay/2 {
			delay = c.reconnectMaxDelay
			break
		}
		delay *= 2
	}
	if delay > c.reconnectMaxDelay {
		delay = c.reconnectMaxDelay
	}

	// Deterministic +/-20% jitter keeps reconnecting workers from synchronizing
	// without introducing a process-global random source into the client.
	const buckets = 41
	bucket := (attempt*17 + 11) % buckets
	percent := 80 + bucket
	jittered := delay * time.Duration(percent) / 100
	if jittered > c.reconnectMaxDelay {
		return c.reconnectMaxDelay
	}
	return jittered
}

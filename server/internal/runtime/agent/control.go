package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"sync"

	"github.com/coder/websocket"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type mediaControl struct {
	connection *websocket.Conn
	events     chan session.Event
	cancel     context.CancelFunc
	closeOnce  sync.Once
}

func (c *mediaClient) OpenControl(ctx context.Context, rawURL string) (*mediaControl, error) {
	if ctx == nil {
		return nil, fmt.Errorf("media control context is required")
	}
	header := http.Header{}
	header.Set("Authorization", "Bearer "+c.token)
	connection, response, err := websocket.Dial(ctx, rawURL, &websocket.DialOptions{
		HTTPHeader:      header,
		CompressionMode: websocket.CompressionDisabled,
	})
	if response != nil && response.Body != nil {
		defer func() { _ = response.Body.Close() }()
	}
	if err != nil {
		if response != nil {
			return nil, fmt.Errorf("connect media control: HTTP %d: %w", response.StatusCode, err)
		}
		return nil, fmt.Errorf("connect media control: %w", err)
	}
	controlCtx, cancel := context.WithCancel(context.Background())
	control := &mediaControl{
		connection: connection,
		events:     make(chan session.Event, 64),
		cancel:     cancel,
	}
	go control.readLoop(controlCtx)
	return control, nil
}

func (c *mediaControl) Events() <-chan session.Event {
	return c.events
}

func (c *mediaControl) Send(ctx context.Context, command session.Command) error {
	if ctx == nil {
		return fmt.Errorf("media control context is required")
	}
	if err := command.Validate(); err != nil {
		return err
	}
	payload, err := json.Marshal(command)
	if err != nil {
		return fmt.Errorf("marshal media command: %w", err)
	}
	if err := c.connection.Write(ctx, websocket.MessageText, payload); err != nil {
		return fmt.Errorf("send media command: %w", err)
	}
	return nil
}

func (c *mediaControl) Close() error {
	var err error
	c.closeOnce.Do(func() {
		c.cancel()
		err = c.connection.Close(websocket.StatusNormalClosure, "control closed")
	})
	return err
}

func (c *mediaControl) readLoop(ctx context.Context) {
	defer close(c.events)
	for {
		kind, payload, err := c.connection.Read(ctx)
		if err != nil {
			return
		}
		if kind != websocket.MessageText {
			return
		}
		var event session.Event
		if err := json.Unmarshal(payload, &event); err != nil {
			return
		}
		select {
		case c.events <- event:
		case <-ctx.Done():
			return
		}
	}
}

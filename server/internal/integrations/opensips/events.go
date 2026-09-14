package opensips

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
)

const DefaultEventPath = "/events"

type EventHandler func(context.Context, Event) error

func (e Event) Validate() error {
	if strings.TrimSpace(e.Name) == "" {
		return fmt.Errorf("OpenSIPS event name is required")
	}
	return nil
}

func (c *Client) Events(ctx context.Context, path string, handler EventHandler) error {
	if err := c.validate(ctx); err != nil {
		return err
	}
	if handler == nil {
		return fmt.Errorf("OpenSIPS event handler is required")
	}

	path = strings.TrimSpace(path)
	if path == "" {
		path = DefaultEventPath
	}
	path = "/" + strings.TrimLeft(path, "/")

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, c.baseURL+path, nil)
	if err != nil {
		return fmt.Errorf("create OpenSIPS event request: %w", err)
	}
	req.Header.Set("Accept", "application/x-ndjson")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("connect to OpenSIPS events: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("OpenSIPS events returned HTTP %d", resp.StatusCode)
	}

	scanner := bufio.NewScanner(resp.Body)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		var event Event
		if err := json.Unmarshal([]byte(line), &event); err != nil {
			return fmt.Errorf("decode OpenSIPS event: %w", err)
		}
		if err := event.Validate(); err != nil {
			return err
		}
		if err := handler(ctx, event); err != nil {
			return fmt.Errorf("handle OpenSIPS event %q: %w", event.Name, err)
		}
	}
	if err := scanner.Err(); err != nil {
		return fmt.Errorf("read OpenSIPS events: %w", err)
	}

	return nil
}

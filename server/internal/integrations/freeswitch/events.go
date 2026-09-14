package freeswitch

import (
	"context"
	"fmt"
	"net/url"
	"strings"
)

type EventHandler func(context.Context, Event) error

type subscription struct {
	format  EventFormat
	events  []string
	handler EventHandler
}

func (s subscription) command() string {
	command := "event " + string(s.format)
	if len(s.events) > 0 {
		command += " " + strings.Join(s.events, " ")
	}
	return command
}

func (s subscription) matches(eventName string) bool {
	if len(s.events) == 0 {
		return true
	}

	for _, event := range s.events {
		if event == eventName {
			return true
		}
	}
	return false
}

func (c *Client) Subscribe(
	ctx context.Context,
	format EventFormat,
	events []string,
	handler EventHandler,
) error {
	subscription, err := newSubscription(format, events, handler)
	if err != nil {
		return err
	}

	c.subscriptionsMu.Lock()
	defer c.subscriptionsMu.Unlock()

	if err := c.sendSubscription(ctx, subscription); err != nil {
		return err
	}
	c.subscriptions = append(c.subscriptions, subscription)

	return nil
}

func newSubscription(
	format EventFormat,
	events []string,
	handler EventHandler,
) (subscription, error) {
	if handler == nil {
		return subscription{}, fmt.Errorf("FreeSWITCH event handler is required")
	}
	if format != EventFormatPlain {
		return subscription{}, fmt.Errorf("unsupported FreeSWITCH event format: %q", format)
	}

	clean := make([]string, 0, len(events))
	seen := make(map[string]struct{}, len(events))
	for _, event := range events {
		event = strings.TrimSpace(event)
		if event == "" {
			continue
		}
		if _, ok := seen[event]; ok {
			continue
		}
		seen[event] = struct{}{}
		clean = append(clean, event)
	}

	return subscription{
		format:  format,
		events:  clean,
		handler: handler,
	}, nil
}

func (c *Client) sendSubscription(ctx context.Context, subscription subscription) error {
	frame, err := c.command(ctx, subscription.command())
	if err != nil {
		return err
	}
	if !frame.OK() {
		return fmt.Errorf("FreeSWITCH event subscription failed: %s", frame.ReplyText())
	}
	return nil
}

func (c *Client) restoreSubscriptions(ctx context.Context) error {
	for _, subscription := range c.subscriptionSnapshot() {
		if err := c.sendSubscription(ctx, subscription); err != nil {
			return err
		}
	}
	return nil
}

func (c *Client) subscriptionSnapshot() []subscription {
	c.subscriptionsMu.RLock()
	defer c.subscriptionsMu.RUnlock()

	return append([]subscription(nil), c.subscriptions...)
}

func plainEventHeader(body, name string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimRight(line, "\r")
		key, value, ok := strings.Cut(line, ": ")
		if !ok || key != name {
			continue
		}
		decoded, err := url.PathUnescape(value)
		if err == nil {
			return decoded
		}
		return value
	}
	return ""
}

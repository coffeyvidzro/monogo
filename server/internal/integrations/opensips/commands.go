package opensips

import (
	"context"
	"fmt"
	"strings"
)

const (
	CommandPing      = "ping"
	CommandReload    = "reload"
	CommandSetFlag   = "set_flag"
	CommandResetFlag = "reset_flag"
)

func (c *Client) Ping(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandPing})
}

func (c *Client) Reload(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandReload})
}

func (c *Client) SetFlag(ctx context.Context, name string) (Response, error) {
	return c.flagCommand(ctx, CommandSetFlag, name)
}

func (c *Client) ResetFlag(ctx context.Context, name string) (Response, error) {
	return c.flagCommand(ctx, CommandResetFlag, name)
}

func (c *Client) flagCommand(ctx context.Context, command, name string) (Response, error) {
	name = strings.TrimSpace(name)
	if name == "" {
		return Response{}, fmt.Errorf("OpenSIPS flag name is required")
	}
	return c.Command(ctx, Command{Name: command, Params: map[string]any{"name": name}})
}

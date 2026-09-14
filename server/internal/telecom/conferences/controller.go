package conferences

import (
	"context"
	"fmt"

	"github.com/leamout/leamout/internal/integrations/freeswitch"
)

// Controller is the media-server contract used by conference controls.
type Controller interface {
	Lock(context.Context, string) error
	Unlock(context.Context, string) error
}

// FreeSWITCHController adapts the FreeSWITCH client to conference controls.
type FreeSWITCHController struct {
	client *freeswitch.Client
}

var _ Controller = (*FreeSWITCHController)(nil)

func NewFreeSWITCHController(client *freeswitch.Client) *FreeSWITCHController {
	if client == nil {
		panic("conferences: FreeSWITCH client is required")
	}
	return &FreeSWITCHController{client: client}
}

func (c *FreeSWITCHController) Lock(ctx context.Context, name string) error {
	if err := c.client.LockConference(ctx, name); err != nil {
		return fmt.Errorf("lock conference: %w", err)
	}
	return nil
}

func (c *FreeSWITCHController) Unlock(ctx context.Context, name string) error {
	if err := c.client.UnlockConference(ctx, name); err != nil {
		return fmt.Errorf("unlock conference: %w", err)
	}
	return nil
}

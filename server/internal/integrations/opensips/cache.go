package opensips

import "context"

const (
	CommandReloadSubscriberCache = "reload_subscriber_cache"
	CommandFlushSubscriberCache  = "flush_subscriber_cache"
)

func (c *Client) ReloadSubscriberCache(ctx context.Context, req SubscriberCacheRequest) (Response, error) {
	params := make(map[string]any, 2)
	if req.Username != "" {
		params["username"] = req.Username
	}
	if req.Domain != "" {
		params["domain"] = req.Domain
	}
	return c.Command(ctx, Command{Name: CommandReloadSubscriberCache, Params: params})
}

func (c *Client) FlushSubscriberCache(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandFlushSubscriberCache})
}

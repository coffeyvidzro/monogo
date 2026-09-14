package opensips

import "context"

const CommandGetStatistics = "get_statistics"

func (c *Client) GetStatistics(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandGetStatistics})
}

func (c *Client) GetActiveCalls(ctx context.Context) (Response, error) {
	return c.metric(ctx, "active_calls")
}

func (c *Client) GetCPS(ctx context.Context) (Response, error) {
	return c.metric(ctx, "cps")
}

func (c *Client) GetRegistrations(ctx context.Context) (Response, error) {
	return c.metric(ctx, "registrations")
}

func (c *Client) metric(ctx context.Context, name string) (Response, error) {
	return c.Command(ctx, Command{Name: CommandGetStatistics, Params: map[string]any{"metric": name}})
}

package opensips

import (
	"context"
	"fmt"
	"strings"
)

const (
	CommandAddRoute    = "add_route"
	CommandRemoveRoute = "remove_route"
	CommandListRoutes  = "list_routes"
)

func (c *Client) AddRoute(ctx context.Context, route Route) (Response, error) {
	if err := validateRoute(route); err != nil {
		return Response{}, err
	}
	return c.Command(ctx, Command{Name: CommandAddRoute, Params: routeParams(route)})
}

func (c *Client) RemoveRoute(ctx context.Context, routeID string) (Response, error) {
	routeID = strings.TrimSpace(routeID)
	if routeID == "" {
		return Response{}, fmt.Errorf("OpenSIPS route ID is required")
	}
	return c.Command(ctx, Command{Name: CommandRemoveRoute, Params: map[string]any{"id": routeID}})
}

func (c *Client) ListRoutes(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandListRoutes})
}

func routeParams(route Route) map[string]any {
	return map[string]any{
		"id":       strings.TrimSpace(route.ID),
		"carrier":  strings.TrimSpace(route.Carrier),
		"prefix":   strings.TrimSpace(route.Prefix),
		"uri":      strings.TrimSpace(route.URI),
		"priority": route.Priority,
		"enabled":  route.Enabled,
	}
}

func validateRoute(route Route) error {
	if strings.TrimSpace(route.ID) == "" {
		return fmt.Errorf("OpenSIPS route ID is required")
	}
	if strings.TrimSpace(route.Carrier) == "" {
		return fmt.Errorf("OpenSIPS route carrier is required")
	}
	if strings.TrimSpace(route.URI) == "" {
		return fmt.Errorf("OpenSIPS route URI is required")
	}
	if route.Priority < 0 {
		return fmt.Errorf("OpenSIPS route priority must not be negative")
	}
	return nil
}

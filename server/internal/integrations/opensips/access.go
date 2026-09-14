package opensips

import (
	"context"
	"fmt"
	"net/netip"
	"strings"
)

const (
	CommandBlockIP        = "block_ip"
	CommandUnblockIP      = "unblock_ip"
	CommandIsBlocked      = "is_blocked"
	CommandListBlockedIPs = "list_blocked_ips"
)

func (c *Client) BlockIP(ctx context.Context, entry AccessEntry) (Response, error) {
	address, err := normalizeIP(entry.Address)
	if err != nil {
		return Response{}, err
	}
	params := map[string]any{"address": address}
	if reason := strings.TrimSpace(entry.Reason); reason != "" {
		params["reason"] = reason
	}
	if !entry.ExpiresAt.IsZero() {
		params["expires_at"] = entry.ExpiresAt.UTC().Format("2006-01-02T15:04:05Z07:00")
	}
	return c.Command(ctx, Command{Name: CommandBlockIP, Params: params})
}

func (c *Client) UnblockIP(ctx context.Context, address string) (Response, error) {
	address, err := normalizeIP(address)
	if err != nil {
		return Response{}, err
	}
	return c.Command(ctx, Command{Name: CommandUnblockIP, Params: map[string]any{"address": address}})
}

func (c *Client) IsBlocked(ctx context.Context, address string) (Response, error) {
	address, err := normalizeIP(address)
	if err != nil {
		return Response{}, err
	}
	return c.Command(ctx, Command{Name: CommandIsBlocked, Params: map[string]any{"address": address}})
}

func (c *Client) ListBlockedIPs(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandListBlockedIPs})
}

func normalizeIP(value string) (string, error) {
	address, err := netip.ParseAddr(strings.TrimSpace(value))
	if err != nil {
		return "", fmt.Errorf("invalid OpenSIPS IP address %q: %w", value, err)
	}
	return address.String(), nil
}

package opensips

import (
	"context"
	"fmt"
	"strings"
)

const (
	CommandListDialogs     = "list_dialogs"
	CommandGetDialog       = "get_dialog"
	CommandTerminateDialog = "terminate_dialog"
)

func (c *Client) ListDialogs(ctx context.Context) (Response, error) {
	return c.Command(ctx, Command{Name: CommandListDialogs})
}

func (c *Client) GetDialog(ctx context.Context, dialogID string) (Response, error) {
	return c.dialogCommand(ctx, CommandGetDialog, dialogID)
}

func (c *Client) TerminateDialog(ctx context.Context, dialogID string) (Response, error) {
	return c.dialogCommand(ctx, CommandTerminateDialog, dialogID)
}

func (c *Client) dialogCommand(ctx context.Context, command, dialogID string) (Response, error) {
	dialogID = strings.TrimSpace(dialogID)
	if dialogID == "" {
		return Response{}, fmt.Errorf("OpenSIPS dialog ID is required")
	}
	return c.Command(ctx, Command{Name: command, Params: map[string]any{"id": dialogID}})
}

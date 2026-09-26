package tools

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
)

const (
	BuiltinHangupCall   = "hangup_call"
	BuiltinTransferCall = "transfer_call"
	BuiltinHoldCall     = "hold_call"
	BuiltinResumeCall   = "resume_call"
	BuiltinSendDTMF     = "send_dtmf"
)

func (e *Executor) executeBuiltin(
	ctx context.Context,
	tool sqlc.VoiceAgentTool,
	req ExecuteRequest,
) (ExecuteResult, error) {
	if e.calls == nil {
		return ExecuteResult{}, fmt.Errorf("Voice Agent built-in call controls are unavailable")
	}

	switch tool.Name {
	case BuiltinHangupCall:
		if err := requireNoBuiltinArguments(req.Arguments); err != nil {
			return ExecuteResult{}, err
		}
		if err := e.calls.Hangup(ctx, req.OrganizationID, req.CallID); err != nil {
			return ExecuteResult{}, err
		}
	case BuiltinTransferCall:
		var input struct {
			Destination string `json:"destination"`
		}
		if err := decodeBuiltinArguments(req.Arguments, &input); err != nil {
			return ExecuteResult{}, err
		}
		if err := e.calls.Transfer(ctx, req.OrganizationID, req.CallID, calls.TransferActionRequest{
			Destination: input.Destination,
		}); err != nil {
			return ExecuteResult{}, err
		}
	case BuiltinHoldCall:
		if err := requireNoBuiltinArguments(req.Arguments); err != nil {
			return ExecuteResult{}, err
		}
		if err := e.calls.Hold(ctx, req.OrganizationID, req.CallID); err != nil {
			return ExecuteResult{}, err
		}
	case BuiltinResumeCall:
		if err := requireNoBuiltinArguments(req.Arguments); err != nil {
			return ExecuteResult{}, err
		}
		if err := e.calls.Resume(ctx, req.OrganizationID, req.CallID); err != nil {
			return ExecuteResult{}, err
		}
	case BuiltinSendDTMF:
		var input struct {
			Digits string `json:"digits"`
		}
		if err := decodeBuiltinArguments(req.Arguments, &input); err != nil {
			return ExecuteResult{}, err
		}
		if err := e.calls.DTMF(ctx, req.OrganizationID, req.CallID, calls.DTMFActionRequest{
			Digits: input.Digits,
		}); err != nil {
			return ExecuteResult{}, err
		}
	default:
		return ExecuteResult{}, fmt.Errorf("unsupported Voice Agent built-in tool %q", tool.Name)
	}

	return ExecuteResult{
		ToolID:      tool.ID,
		ToolCallID:  req.ToolCallID,
		Name:        tool.Name,
		StatusCode:  200,
		ContentType: "application/json",
		Body:        []byte(`{"ok":true}`),
	}, nil
}

func decodeBuiltinArguments(arguments json.RawMessage, dst any) error {
	decoder := json.NewDecoder(bytes.NewReader(arguments))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(dst); err != nil {
		return fmt.Errorf("invalid built-in tool arguments: %w", err)
	}
	return nil
}

func requireNoBuiltinArguments(arguments json.RawMessage) error {
	var input map[string]any
	if err := json.Unmarshal(arguments, &input); err != nil {
		return fmt.Errorf("invalid built-in tool arguments: %w", err)
	}
	if len(input) != 0 {
		return fmt.Errorf("built-in tool does not accept arguments")
	}
	return nil
}

package calls

import (
	"context"

	"github.com/google/uuid"
)

// Controller is the runtime boundary for executing call-control operations.
// Implementations belong in runtime/integration packages; the calls domain only
// depends on this contract.
type Controller interface {
	Originate(context.Context, OriginateCommand) (OriginateResult, error)
	Hangup(context.Context, string) error
	Hold(context.Context, string) error
	Resume(context.Context, string) error
}

type OriginateCommand struct {
	CallID  uuid.UUID
	FromURI string
	ToURI   string
}

type OriginateResult struct {
	ChannelID  string
	SIPCallID  string
	ProviderID *uuid.UUID
}

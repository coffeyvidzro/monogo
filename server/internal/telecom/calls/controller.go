package calls

import (
	"context"

	"github.com/google/uuid"
)

// Controller is the runtime boundary for executing call-control operations.
// Runtime adapters implement this interface without leaking FreeSWITCH details
// into the calls domain.
type Controller interface {
	Originate(context.Context, OriginateRequest) (OriginateResult, error)
	Answer(context.Context, string) error
	Hangup(context.Context, string) error
	Transfer(context.Context, string, TransferRequest) error
	Hold(context.Context, string) error
	Resume(context.Context, string) error
	PlayAudio(context.Context, string, string) error
	StopPlayback(context.Context, string) error
	Record(context.Context, string, RecordRequest) error
	SendDTMF(context.Context, string, string) error
}

type OriginateRequest struct {
	CallID              uuid.UUID
	Destination         string
	CallerID            string
	CarrierConnectionID uuid.UUID
	Host                string
	Port                uint16
	Transport           string
	Privacy             bool
	DTMFMode            string
	MediaEncryption     string
}

type OriginateResult struct {
	ChannelID string
}

type TransferRequest struct {
	Destination string
	Dialplan    string
	Context     string
}

type RecordRequest struct {
	Path   string
	Action string
}

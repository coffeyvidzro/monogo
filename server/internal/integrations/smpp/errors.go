package smpp

import "errors"

var (
	ErrNotConnected       = errors.New("SMPP session is not connected")
	ErrSubmitUnsupported  = errors.New("SMPP receiver bind cannot submit messages")
	ErrMalformedDeliverSM = errors.New("malformed SMPP deliver_sm")
	ErrMalformedReceipt   = errors.New("malformed SMPP delivery receipt")
)

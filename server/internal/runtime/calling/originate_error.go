package calling

import (
	"errors"
	"fmt"
)

type OriginateFailureClass string

const (
	OriginateFailureValidation OriginateFailureClass = "validation"
	OriginateFailureTransport  OriginateFailureClass = "transport"
	OriginateFailureTimeout    OriginateFailureClass = "timeout"
	OriginateFailureSIP        OriginateFailureClass = "sip"
	OriginateFailureCapacity   OriginateFailureClass = "capacity"
	OriginateFailureInternal   OriginateFailureClass = "internal"
)

// OriginateError carries the information required to make a failover decision
// without parsing provider or FreeSWITCH error strings.
type OriginateError struct {
	Class     OriginateFailureClass
	SIPStatus int
	Err       error
}

func (e *OriginateError) Error() string {
	if e == nil {
		return "originate failure"
	}
	if e.SIPStatus != 0 {
		return fmt.Sprintf("originate failure (%s, SIP %d): %v", e.Class, e.SIPStatus, e.Err)
	}
	return fmt.Sprintf("originate failure (%s): %v", e.Class, e.Err)
}

func (e *OriginateError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.Err
}

func NewSIPOriginateError(status int, err error) error {
	if err == nil {
		err = errors.New("SIP originate failed")
	}
	return &OriginateError{Class: OriginateFailureSIP, SIPStatus: status, Err: err}
}

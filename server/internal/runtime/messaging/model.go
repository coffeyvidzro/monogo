package messaging

import (
	"fmt"

	"github.com/google/uuid"
)

type Channel string

const (
	ChannelSMS      Channel = "sms"
	ChannelWhatsApp Channel = "whatsapp"
)

type Request struct {
	MessageID, ConnectionID uuid.UUID
	Channel                 Channel
	From, To, Text          string
}

type Result struct {
	ExternalID string
}

type SubmissionOutcome string

const (
	SubmissionNotSubmitted SubmissionOutcome = "not_submitted"
	SubmissionUnknown      SubmissionOutcome = "unknown"
	SubmissionRejected     SubmissionOutcome = "rejected"
)

type SubmissionError struct {
	Outcome SubmissionOutcome
	Err     error
}

func (e *SubmissionError) Error() string {
	return fmt.Sprintf("%s: %v", e.Outcome, e.Err)
}

func (e *SubmissionError) Unwrap() error {
	return e.Err
}

func newSubmissionError(outcome SubmissionOutcome, err error) error {
	return &SubmissionError{Outcome: outcome, Err: err}
}

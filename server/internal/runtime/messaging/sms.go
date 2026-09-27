package messaging

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
)

type SMS struct{ client *smpp.Client }

func NewSMS(client *smpp.Client) *SMS {
	return &SMS{
		client: client,
	}
}

func (s *SMS) Send(ctx context.Context, request Request) (Result, error) {
	result, err := s.client.Submit(ctx, smpp.SubmitRequest{
		From:                   request.From,
		To:                     request.To,
		Text:                   request.Text,
		RequestDeliveryReceipt: true,
	})
	if err != nil {
		switch {
		case errors.Is(err, smpp.ErrNotConnected):
			return Result{}, newSubmissionError(SubmissionNotSubmitted, err)
		case errors.Is(err, smpp.ErrSubmitUnsupported):
			return Result{}, newSubmissionError(SubmissionRejected, err)
		default:
			return Result{}, newSubmissionError(SubmissionUnknown, err)
		}
	}
	return Result{
		ExternalID: result.MessageID,
	}, nil
}

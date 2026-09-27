package messaging

import (
	"context"

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
		return Result{}, err
	}
	return Result{
		ExternalID: result.MessageID,
	}, nil
}

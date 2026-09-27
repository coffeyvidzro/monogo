package messaging

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
)

type SMPPClient interface {
	Submit(context.Context, smpp.SubmitRequest) (smpp.SubmitResult, error)
}

// SMS translates canonical runtime requests to the direct-carrier SMPP API.
type SMS struct{ client SMPPClient }

func NewSMS(client SMPPClient) *SMS { return &SMS{client: client} }

func (s *SMS) Send(ctx context.Context, request Request) (Result, error) {
	result, err := s.client.Submit(ctx, smpp.SubmitRequest{From: request.From, To: request.To, Text: request.Text})
	if err != nil {
		return Result{}, err
	}
	return Result{ExternalID: result.MessageID}, nil
}

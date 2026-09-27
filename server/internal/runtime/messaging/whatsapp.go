package messaging

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
)

type WhatsAppClient interface {
	SendMessage(context.Context, whatsapp.MessageRequest) (whatsapp.MessageResult, error)
}

// WhatsApp translates canonical runtime requests to the Business Platform API.
type WhatsApp struct{ client WhatsAppClient }

func NewWhatsApp(client WhatsAppClient) *WhatsApp { return &WhatsApp{client: client} }

func (w *WhatsApp) Send(ctx context.Context, request Request) (Result, error) {
	result, err := w.client.SendMessage(ctx, whatsapp.MessageRequest{To: request.To, Text: request.Text})
	if err != nil {
		return Result{}, err
	}
	return Result{ExternalID: result.MessageID}, nil
}

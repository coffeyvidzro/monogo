package messaging

import (
	"context"
	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
)

type WhatsApp struct{ client *whatsapp.Client }

func NewWhatsApp(client *whatsapp.Client) *WhatsApp { return &WhatsApp{client: client} }
func (w *WhatsApp) Send(ctx context.Context, request Request) (Result, error) {
	result, err := w.client.SendMessage(ctx, whatsapp.MessageRequest{To: request.To, Text: request.Text})
	if err != nil {
		return Result{}, err
	}
	return Result{ExternalID: result.MessageID}, nil
}

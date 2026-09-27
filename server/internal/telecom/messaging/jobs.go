package messaging

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
)

// Jobs consumes protocol events and applies canonical lifecycle transitions.
type Jobs struct{ service *Service }

func NewJobs(service *Service) *Jobs {
	if service == nil {
		panic("messaging: service is required")
	}
	return &Jobs{service: service}
}
func (j *Jobs) RunSMPP(ctx context.Context, connection sqlc.MessagingConnection, client *smpp.Client) error {
	if client == nil {
		return fmt.Errorf("SMPP client is required")
	}
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case event := <-client.Inbound():
			body := event.Text
			_, err := j.service.AcceptInbound(ctx, InboundRequest{OrganizationID: connection.OrganizationID, MessagingConnectionID: connection.ID, ProviderMessageID: event.Reference, Channel: ChannelSMS, From: event.From, To: event.To, Body: &body})
			if err != nil {
				return err
			}
		case receipt := <-client.Deliveries():
			status := DeliveryUndelivered
			if receipt.State == smpp.DeliveryDelivered {
				status = DeliveryDelivered
			} else if receipt.State == smpp.DeliveryAccepted || receipt.State == smpp.DeliveryEnroute {
				status = DeliverySent
			}
			code := receipt.ErrorCode
			_, err := j.service.AcceptDelivery(ctx, DeliveryReceipt{MessagingConnectionID: connection.ID, ProviderMessageID: receipt.MessageID, Status: status, FailureCode: &code})
			if err != nil {
				return err
			}
		case err := <-client.Errors():
			return err
		}
	}
}

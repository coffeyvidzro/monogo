package messaging

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
)

// Jobs consumes protocol events and applies canonical lifecycle transitions.
type Jobs struct {
	service *Service
	repo    *Repository
}

func NewJobs(service *Service, repo *Repository) *Jobs {
	if service == nil || repo == nil {
		panic("messaging: service and repository are required")
	}
	return &Jobs{
		service: service,
		repo:    repo,
	}
}

func (j *Jobs) RunSubmissionReconciliation(ctx context.Context) error {
	ticker := time.NewTicker(time.Minute)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return nil
		case now := <-ticker.C:
			if err := j.service.ReconcileStaleSubmissions(ctx, now.UTC().Add(-5*time.Minute)); err != nil {
				return err
			}
		}
	}
}
func (j *Jobs) RunSMPP(ctx context.Context, connection sqlc.MessagingConnection, client *smpp.Client) error {
	if client == nil {
		return fmt.Errorf("SMPP client is required")
	}
	for {
		select {
		case <-ctx.Done():
			return nil
		case event := <-client.Inbound():
			organizationID, err := j.repo.ResolveInboundOrganization(ctx, connection.ID, event.To)
			if err != nil {
				return fmt.Errorf("resolve inbound SMS organization: %w", err)
			}
			body := event.Text
			_, err = j.service.AcceptInbound(ctx, InboundRequest{
				OrganizationID:        organizationID,
				MessagingConnectionID: connection.ID,
				Channel:               ChannelSMS,
				From:                  event.From,
				To:                    event.To,
				Body:                  &body,
			})
			if err != nil {
				return err
			}
		case receipt := <-client.Deliveries():
			status := DeliveryUndelivered
			switch receipt.State {
			case smpp.DeliveryDelivered:
				status = DeliveryDelivered
			case smpp.DeliveryAccepted, smpp.DeliveryEnroute:
				status = DeliverySent
			}
			code := receipt.ErrorCode
			_, err := j.service.AcceptDelivery(ctx, DeliveryReceipt{
				MessagingConnectionID: connection.ID,
				ProviderMessageID:     receipt.MessageID,
				Status:                status,
				FailureCode:           &code,
			})
			if err != nil {
				return err
			}
		case err := <-client.Errors():
			return err
		}
	}
}

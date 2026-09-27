package messaging

import (
	"context"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
)

// DeliveryStatus is the carrier-neutral subset of delivery receipt states.
type DeliveryStatus string

const (
	DeliverySent        DeliveryStatus = "sent"
	DeliveryDelivered   DeliveryStatus = "delivered"
	DeliveryUndelivered DeliveryStatus = "undelivered"
	DeliveryFailed      DeliveryStatus = "failed"
)

// DeliveryReceipt is produced by a carrier adapter after it has authenticated
// and normalized a provider callback.
type DeliveryReceipt struct {
	MessagingConnectionID uuid.UUID
	ProviderMessageID     string
	Status                DeliveryStatus
	FailureCode           *string
	FailureMessage        *string
}

// AcceptInbound persists a normalized inbound provider callback. The database
// uniqueness key makes repeated provider deliveries idempotent.
func (s *Service) AcceptInbound(ctx context.Context, req InboundRequest) (sqlc.Message, error) {
	return s.CreateInbound(ctx, req)
}

// AcceptDelivery applies a normalized delivery receipt. Replayed receipts and
// stale intermediate states are acknowledged by returning the current message.
func (s *Service) AcceptDelivery(ctx context.Context, receipt DeliveryReceipt) (sqlc.Message, error) {
	receipt.ProviderMessageID = strings.TrimSpace(receipt.ProviderMessageID)
	if receipt.MessagingConnectionID == uuid.Nil {
		return sqlc.Message{}, apperror.NewBadRequest("messaging_connection_id is required")
	}
	if receipt.ProviderMessageID == "" || len(receipt.ProviderMessageID) > 255 {
		return sqlc.Message{}, apperror.NewBadRequest("provider_message_id must be between 1 and 255 characters")
	}
	if !receipt.Status.IsValid() {
		return sqlc.Message{}, apperror.NewBadRequest("invalid delivery status")
	}

	message, err := s.repo.GetByProviderID(ctx, receipt.MessagingConnectionID, receipt.ProviderMessageID)
	if err != nil {
		return sqlc.Message{}, messageReadError(err, "message not found")
	}
	if deliveryAlreadyAccepted(Status(message.Status), receipt.Status) {
		return message, nil
	}

	switch receipt.Status {
	case DeliverySent:
		return s.MarkSent(ctx, message.OrganizationID, message.ID)
	case DeliveryDelivered:
		return s.MarkDelivered(ctx, message.OrganizationID, message.ID)
	case DeliveryUndelivered:
		return s.MarkUndelivered(ctx, message.OrganizationID, message.ID, Failure{
			Code: receipt.FailureCode, Message: receipt.FailureMessage,
		})
	case DeliveryFailed:
		return s.MarkFailed(ctx, message.OrganizationID, message.ID, Failure{
			Code: receipt.FailureCode, Message: receipt.FailureMessage,
		})
	default:
		panic("validated delivery status was not handled")
	}
}

func (s DeliveryStatus) IsValid() bool {
	switch s {
	case DeliverySent, DeliveryDelivered, DeliveryUndelivered, DeliveryFailed:
		return true
	default:
		return false
	}
}

func deliveryAlreadyAccepted(current Status, incoming DeliveryStatus) bool {
	if string(current) == string(incoming) {
		return true
	}
	// Terminal provider outcomes are immutable. A delayed "sent" receipt after
	// delivery is stale, and any receipt after a terminal failure is also stale.
	if current == StatusDelivered || current == StatusUndelivered || current == StatusFailed {
		return true
	}
	return current == StatusSent && incoming == DeliverySent
}

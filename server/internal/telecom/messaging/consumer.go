package messaging

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	runtimemessaging "github.com/coffeyvidzro/monogo/internal/runtime/messaging"
)

// Consumer owns canonical state transitions and delegates execution to the runtime.
type Consumer struct {
	service *Service
	repo    *Repository
	runtime *runtimemessaging.Controller
}

func NewConsumer(service *Service, repo *Repository, runtime *runtimemessaging.Controller) *Consumer {
	if service == nil || repo == nil || runtime == nil {
		panic("messaging: service, repository, and runtime are required")
	}
	return &Consumer{service: service, repo: repo, runtime: runtime}
}
func (c *Consumer) HandleQueued(ctx context.Context, message sqlc.Message) (sqlc.Message, error) {
	if message.Direction != string(DirectionOutbound) || message.Status != string(StatusQueued) {
		return sqlc.Message{}, fmt.Errorf("message %s is not queued outbound", message.ID)
	}
	connection, err := c.repo.ResolveConnection(ctx, message.OrganizationID, Channel(message.Channel))
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("resolve messaging connection: %w", err)
	}
	attributed, err := c.service.SetProviderAttribution(ctx, message.OrganizationID, message.ID, connection.ID)
	if err != nil {
		return sqlc.Message{}, err
	}
	request, err := runtimemessaging.Translate(attributed.ID, connection.ID, runtimemessaging.Channel(attributed.Channel), attributed.FromAddress, attributed.ToAddress, attributed.Body)
	if err != nil {
		return sqlc.Message{}, err
	}
	result, err := c.runtime.Send(ctx, request)
	if err != nil {
		return sqlc.Message{}, fmt.Errorf("submit %s message: %w", message.Channel, err)
	}
	providerID := strings.TrimSpace(result.ExternalID)
	if providerID == "" {
		code, detail := "invalid_provider_response", "transport accepted message without an id"
		return c.service.MarkFailed(ctx, message.OrganizationID, message.ID, Failure{Code: &code, Message: &detail})
	}
	return c.service.MarkSubmitted(ctx, message.OrganizationID, message.ID, providerID)
}
func unmarshalMedia(payload []byte, target *[]Media) error {
	if len(payload) == 0 {
		*target = []Media{}
		return nil
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode message media: %w", err)
	}
	return nil
}

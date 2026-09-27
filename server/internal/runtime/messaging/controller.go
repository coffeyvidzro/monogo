// Package messaging executes canonical messages through configured channel integrations.
package messaging

import (
	"context"
	"fmt"
	"sync"

	"github.com/coffeyvidzro/monogo/internal/integrations/smpp"
	"github.com/coffeyvidzro/monogo/internal/integrations/whatsapp"
	"github.com/google/uuid"
)

type Controller struct {
	mu       sync.RWMutex
	sms      map[uuid.UUID]*SMS
	whatsapp map[uuid.UUID]*WhatsApp
}

func NewController() *Controller {
	return &Controller{
		sms:      make(map[uuid.UUID]*SMS),
		whatsapp: make(map[uuid.UUID]*WhatsApp),
	}
}
func (c *Controller) RegisterSMS(id uuid.UUID, client *smpp.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.sms[id] = NewSMS(client)
}
func (c *Controller) RegisterWhatsApp(id uuid.UUID, client *whatsapp.Client) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.whatsapp[id] = NewWhatsApp(client)
}
func (c *Controller) Send(ctx context.Context, request Request) (Result, error) {
	if request.ConnectionID == uuid.Nil {
		return Result{}, fmt.Errorf("messaging connection is required")
	}
	c.mu.RLock()
	defer c.mu.RUnlock()
	switch request.Channel {
	case ChannelSMS:
		adapter := c.sms[request.ConnectionID]
		if adapter == nil {
			return Result{}, fmt.Errorf("SMS connection %s is not registered", request.ConnectionID)
		}
		return adapter.Send(ctx, request)
	case ChannelWhatsApp:
		adapter := c.whatsapp[request.ConnectionID]
		if adapter == nil {
			return Result{}, fmt.Errorf("WhatsApp connection %s is not registered", request.ConnectionID)
		}
		return adapter.Send(ctx, request)
	default:
		return Result{}, fmt.Errorf("unsupported messaging channel %q", request.Channel)
	}
}

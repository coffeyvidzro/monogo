package media

import (
	"context"
	"encoding/json"
	"fmt"

	natsintegration "github.com/coffeyvidzro/monogo/internal/integrations/nats"
	"github.com/coffeyvidzro/monogo/internal/media/session"
)

type eventPublisher struct {
	nats *natsintegration.Client
}

func (p *eventPublisher) Publish(ctx context.Context, event session.PublishedEvent) error {
	payload, err := json.Marshal(event)
	if err != nil {
		return fmt.Errorf("marshal media event: %w", err)
	}
	messageID := fmt.Sprintf("%s:%d", event.SessionID, event.Generation)
	return p.nats.PublishWithOptions(
		ctx,
		natsintegration.VoiceAgentMediaEventSubject,
		payload,
		map[string]string{
			"Leamout-Session-ID": event.SessionID.String(),
		},
		messageID,
	)
}

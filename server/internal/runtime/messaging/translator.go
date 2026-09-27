package messaging

import (
	"fmt"
	"strings"

	domain "github.com/coffeyvidzro/monogo/internal/telecom/messaging"
	"github.com/google/uuid"
)

func Translate(route domain.Route, message domain.OutboundMessage) (Request, error) {
	if route.CarrierConnectionID == uuid.Nil {
		return Request{}, fmt.Errorf("carrier connection is required")
	}
	if message.MessageID == uuid.Nil {
		return Request{}, fmt.Errorf("message id is required")
	}
	text := ""
	if message.Body != nil {
		text = *message.Body
	}
	return Request{MessageID: message.MessageID, ConnectionID: route.CarrierConnectionID, From: strings.TrimSpace(message.From), To: strings.TrimSpace(message.To), Text: text}, nil
}

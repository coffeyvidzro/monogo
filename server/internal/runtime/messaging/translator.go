package messaging

import (
	"fmt"
	"github.com/google/uuid"
	"strings"
)

func Translate(messageID, connectionID uuid.UUID, channel Channel, from, to string, body *string) (Request, error) {
	if messageID == uuid.Nil || connectionID == uuid.Nil {
		return Request{}, fmt.Errorf("message and connection ids are required")
	}
	text := ""
	if body != nil {
		text = *body
	}
	return Request{
		MessageID:    messageID,
		ConnectionID: connectionID,
		Channel:      channel,
		From:         strings.TrimSpace(from),
		To:           strings.TrimSpace(to),
		Text:         text,
	}, nil
}

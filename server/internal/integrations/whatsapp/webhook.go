package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strconv"
	"strings"
)

func (c *Client) VerifySignature(body []byte, signature string) bool {
	if c == nil || c.config.AppSecret == "" {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.config.AppSecret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

// ParseWebhook normalizes official messages and statuses webhook changes.
func ParseWebhook(body []byte) ([]WebhookEvent, error) {
	var payload webhookPayload
	if err := json.Unmarshal(body, &payload); err != nil {
		return nil, fmt.Errorf("decode WhatsApp webhook: %w", err)
	}
	if payload.Object != "whatsapp_business_account" {
		return nil, fmt.Errorf("unexpected WhatsApp webhook object %q", payload.Object)
	}
	events := make([]WebhookEvent, 0)
	for _, entry := range payload.Entry {
		for _, change := range entry.Changes {
			if change.Field != "messages" {
				continue
			}
			v := change.Value
			for _, message := range v.Messages {
				if message.Type != "text" {
					continue
				}
				events = append(events, WebhookEvent{Kind: EventMessage, PhoneNumberID: v.Metadata.PhoneNumberID, MessageID: message.ID, From: message.From, To: v.Metadata.PhoneNumberID, Text: message.Text.Body})
			}
			for _, status := range v.Statuses {
				event := WebhookEvent{Kind: EventStatus, PhoneNumberID: v.Metadata.PhoneNumberID, MessageID: status.ID, To: status.RecipientID, Status: status.Status}
				if len(status.Errors) > 0 {
					event.ErrorCode = strconv.Itoa(status.Errors[0].Code)
					event.ErrorMessage = status.Errors[0].Message
				}
				events = append(events, event)
			}
		}
	}
	return events, nil
}

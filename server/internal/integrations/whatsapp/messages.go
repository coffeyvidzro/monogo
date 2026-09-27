package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
)

// SendMessage exposes WhatsApp-owned types; canonical translation belongs to runtime/messaging.
func (c *Client) SendMessage(ctx context.Context, message MessageRequest) (MessageResult, error) {
	payload, err := json.Marshal(sendRequest{
		MessagingProduct: "whatsapp",
		To:               message.To,
		Type:             "text",
		Text: textPayload{
			Body: message.Text,
		},
	})
	if err != nil {
		return MessageResult{}, fmt.Errorf("encode WhatsApp message: %w", err)
	}
	url := c.config.BaseURL + "/" + c.config.PhoneNumberID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return MessageResult{}, fmt.Errorf("create WhatsApp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return MessageResult{}, fmt.Errorf("send WhatsApp message: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return MessageResult{}, fmt.Errorf("read WhatsApp response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return MessageResult{}, &APIError{
			Status:  resp.StatusCode,
			Message: string(body),
		}
	}
	var result sendResponse
	if err := json.Unmarshal(body, &result); err != nil || len(result.Messages) == 0 || result.Messages[0].ID == "" {
		return MessageResult{}, fmt.Errorf("invalid WhatsApp message response")
	}
	return MessageResult{
		MessageID: result.Messages[0].ID,
	}, nil
}

package whatsapp

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	runtime "github.com/coffeyvidzro/monogo/internal/runtime/messaging"
)

func (c *Client) Send(ctx context.Context, message runtime.Request) (runtime.Result, error) {
	payload, err := json.Marshal(sendRequest{MessagingProduct: "whatsapp", To: message.To, Type: "text", Text: textPayload{Body: message.Text}})
	if err != nil {
		return runtime.Result{}, fmt.Errorf("encode WhatsApp message: %w", err)
	}
	url := c.config.BaseURL + "/" + c.config.PhoneNumberID + "/messages"
	req, err := http.NewRequestWithContext(ctx, http.MethodPost, url, bytes.NewReader(payload))
	if err != nil {
		return runtime.Result{}, fmt.Errorf("create WhatsApp request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.config.AccessToken)
	req.Header.Set("Content-Type", "application/json")
	resp, err := c.http.Do(req)
	if err != nil {
		return runtime.Result{}, fmt.Errorf("send WhatsApp message: %w", err)
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return runtime.Result{}, fmt.Errorf("read WhatsApp response: %w", err)
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return runtime.Result{}, &APIError{Status: resp.StatusCode, Message: string(body)}
	}
	var result sendResponse
	if err := json.Unmarshal(body, &result); err != nil || len(result.Messages) == 0 || result.Messages[0].ID == "" {
		return runtime.Result{}, fmt.Errorf("invalid WhatsApp message response")
	}
	return runtime.Result{ExternalID: result.Messages[0].ID}, nil
}

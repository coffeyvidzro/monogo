package webhooks

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

const maxResponseBody = 4096

type DeliverySender interface {
	Send(context.Context, sqlc.ClaimWebhookDeliveriesRow) DeliveryAttempt
}

type HTTPSender struct {
	client *http.Client
}

func NewHTTPSender() *HTTPSender {
	return &HTTPSender{client: &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}}
}

func (s *HTTPSender) Send(ctx context.Context, delivery sqlc.ClaimWebhookDeliveriesRow) DeliveryAttempt {
	body, err := json.Marshal(DeliveryEnvelope{
		ID:         delivery.EventID,
		Type:       delivery.EventType,
		OccurredAt: pgconv.TimestamptzToTime(delivery.OccurredAt),
		Data:       delivery.Payload,
	})
	if err != nil {
		return DeliveryAttempt{Err: fmt.Errorf("marshal webhook delivery: %w", err)}
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, delivery.Url, bytes.NewReader(body))
	if err != nil {
		return DeliveryAttempt{Err: fmt.Errorf("create webhook request: %w", err)}
	}
	now := time.Now().UTC()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Leamout-Webhooks/1.0")
	req.Header.Set("X-Leamout-Event", delivery.EventType)
	req.Header.Set("X-Leamout-Event-ID", delivery.EventID.String())
	req.Header.Set("X-Leamout-Timestamp", fmt.Sprintf("%d", now.Unix()))
	req.Header.Set(signatureHeader, sign(delivery.SigningSecret, body, now))

	resp, err := s.client.Do(req)
	if err != nil {
		return DeliveryAttempt{Err: err}
	}
	defer func() { _ = resp.Body.Close() }()

	responseBody, readErr := io.ReadAll(io.LimitReader(resp.Body, maxResponseBody))
	status := int32(resp.StatusCode)
	bodyText := string(responseBody)
	attempt := DeliveryAttempt{StatusCode: &status, Body: &bodyText}
	if readErr != nil {
		attempt.Err = fmt.Errorf("read webhook response: %w", readErr)
	}
	return attempt
}

func sendTest(c context.Context, endpoint sqlc.WebhookEndpoint) (int, error) {
	body, err := json.Marshal(map[string]any{"id": "test", "type": "webhook.test", "occurred_at": time.Now().UTC(), "data": map[string]bool{"test": true}})
	if err != nil {
		return 0, err
	}
	req, err := http.NewRequestWithContext(c, http.MethodPost, endpoint.Url, bytes.NewReader(body))
	if err != nil {
		return 0, err
	}
	now := time.Now().UTC()
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Leamout-Webhooks/1.0")
	req.Header.Set("X-Leamout-Event", "webhook.test")
	req.Header.Set("X-Leamout-Timestamp", fmt.Sprintf("%d", now.Unix()))
	req.Header.Set(signatureHeader, sign(endpoint.SigningSecret, body, now))
	client := &http.Client{
		Timeout: 10 * time.Second,
		CheckRedirect: func(_ *http.Request, _ []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
	resp, err := client.Do(req)
	if err != nil {
		return 0, err
	}
	defer func() { _ = resp.Body.Close() }()
	_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, maxResponseBody))
	return resp.StatusCode, nil
}

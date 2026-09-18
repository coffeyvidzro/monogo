package paystack

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha512"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

type Client struct {
	secretKey  string
	baseURL    string
	httpClient *http.Client
}

type apiResponse[T any] struct {
	Status  bool   `json:"status"`
	Message string `json:"message"`
	Data    T      `json:"data"`
}

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(cfg.BaseURL) == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = DefaultConfig(cfg.SecretKey).HTTPClient
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}

	return &Client{
		secretKey:  strings.TrimSpace(cfg.SecretKey),
		baseURL:    strings.TrimRight(cfg.BaseURL, "/"),
		httpClient: cfg.HTTPClient,
	}, nil
}

func (c *Client) ChargeMobileMoney(ctx context.Context, request ChargeRequest) (Charge, error) {
	if err := request.Validate(); err != nil {
		return Charge{}, err
	}

	var response apiResponse[Charge]
	if err := c.doJSON(ctx, http.MethodPost, "/charge", request, &response); err != nil {
		return Charge{}, err
	}
	if !response.Status {
		return Charge{}, fmt.Errorf("Paystack charge failed: %s", response.Message)
	}

	return response.Data, nil
}

func (c *Client) VerifyTransaction(ctx context.Context, reference string) (Transaction, error) {
	reference = strings.TrimSpace(reference)
	if reference == "" {
		return Transaction{}, fmt.Errorf("Paystack reference is required")
	}

	var response apiResponse[Transaction]
	if err := c.doJSON(
		ctx,
		http.MethodGet,
		"/transaction/verify/"+url.PathEscape(reference),
		nil,
		&response,
	); err != nil {
		return Transaction{}, err
	}
	if !response.Status {
		return Transaction{}, fmt.Errorf("Paystack verification failed: %s", response.Message)
	}

	return response.Data, nil
}

func (c *Client) ParseWebhook(payload []byte, signature string) (WebhookEvent, error) {
	signature = strings.TrimSpace(signature)
	if signature == "" {
		return WebhookEvent{}, fmt.Errorf("Paystack webhook signature is required")
	}

	mac := hmac.New(sha512.New, []byte(c.secretKey))
	_, _ = mac.Write(payload)
	expected := hex.EncodeToString(mac.Sum(nil))
	if !hmac.Equal([]byte(expected), []byte(strings.ToLower(signature))) {
		return WebhookEvent{}, fmt.Errorf("invalid Paystack webhook signature")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return WebhookEvent{}, fmt.Errorf("decode Paystack webhook: %w", err)
	}
	if strings.TrimSpace(event.Event) == "" {
		return WebhookEvent{}, fmt.Errorf("Paystack webhook event type is required")
	}
	if strings.TrimSpace(event.Data.Reference) == "" {
		return WebhookEvent{}, fmt.Errorf("Paystack webhook reference is required")
	}

	return event, nil
}

func (c *Client) doJSON(
	ctx context.Context,
	method, path string,
	body any,
	target any,
) error {
	if ctx == nil {
		return fmt.Errorf("Paystack context is required")
	}

	var reader io.Reader
	if body != nil {
		payload, err := json.Marshal(body)
		if err != nil {
			return fmt.Errorf("encode Paystack request: %w", err)
		}
		reader = bytes.NewReader(payload)
	}

	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, reader)
	if err != nil {
		return fmt.Errorf("create Paystack request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Accept", "application/json")
	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Paystack request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read Paystack response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf(
			"Paystack request failed with status %d: %s",
			resp.StatusCode,
			strings.TrimSpace(string(payload)),
		)
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode Paystack response: %w", err)
	}

	return nil
}

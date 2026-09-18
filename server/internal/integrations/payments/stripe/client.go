package stripe

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"
)

const webhookTolerance = 5 * time.Minute

type Client struct {
	secretKey     string
	webhookSecret string
	baseURL       string
	httpClient    *http.Client
}

func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	if cfg.HTTPClient == nil {
		cfg.HTTPClient = DefaultConfig(cfg.SecretKey, cfg.WebhookSecret).HTTPClient
	}
	if err := cfg.Validate(); err != nil {
		return nil, err
	}
	return &Client{
		secretKey:     cfg.SecretKey,
		webhookSecret: cfg.WebhookSecret,
		baseURL:       strings.TrimRight(cfg.BaseURL, "/"),
		httpClient:    cfg.HTTPClient,
	}, nil
}

func (c *Client) CreateCheckoutSession(
	ctx context.Context,
	request CreateCheckoutSessionRequest,
) (CheckoutSession, error) {
	if err := request.Validate(); err != nil {
		return CheckoutSession{}, err
	}

	values := url.Values{}
	values.Set("mode", "payment")
	values.Set("ui_mode", "custom")
	values.Set("payment_method_types[0]", "card")
	values.Set("line_items[0][quantity]", "1")
	values.Set("line_items[0][price_data][currency]", strings.ToLower(request.Currency))
	values.Set("line_items[0][price_data][unit_amount]", strconv.FormatInt(request.Amount, 10))
	values.Set("line_items[0][price_data][product_data][name]", "Leamout")

	var session CheckoutSession
	if err := c.doForm(ctx, http.MethodPost, "/checkout/sessions", values, &session); err != nil {
		return CheckoutSession{}, err
	}
	return session, nil
}

func (c *Client) RetrieveCheckoutSession(ctx context.Context, id string) (CheckoutSession, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return CheckoutSession{}, fmt.Errorf("Stripe checkout session ID is required")
	}

	var session CheckoutSession
	if err := c.doForm(
		ctx,
		http.MethodGet,
		"/checkout/sessions/"+url.PathEscape(id),
		nil,
		&session,
	); err != nil {
		return CheckoutSession{}, err
	}
	return session, nil
}

func (c *Client) ParseWebhook(payload []byte, signatureHeader string, now time.Time) (WebhookEvent, error) {
	if strings.TrimSpace(c.webhookSecret) == "" {
		return WebhookEvent{}, fmt.Errorf("Stripe webhook secret is required")
	}
	timestamp, signatures, err := parseSignatureHeader(signatureHeader)
	if err != nil {
		return WebhookEvent{}, err
	}
	if now.IsZero() {
		now = time.Now()
	}
	eventTime := time.Unix(timestamp, 0)
	if now.Sub(eventTime) > webhookTolerance || eventTime.Sub(now) > webhookTolerance {
		return WebhookEvent{}, fmt.Errorf("Stripe webhook timestamp is outside tolerance")
	}

	signedPayload := strconv.FormatInt(timestamp, 10) + "." + string(payload)
	mac := hmac.New(sha256.New, []byte(c.webhookSecret))
	_, _ = mac.Write([]byte(signedPayload))
	expected := mac.Sum(nil)

	valid := false
	for _, signature := range signatures {
		decoded, decodeErr := hex.DecodeString(signature)
		if decodeErr == nil && hmac.Equal(expected, decoded) {
			valid = true
			break
		}
	}
	if !valid {
		return WebhookEvent{}, fmt.Errorf("invalid Stripe webhook signature")
	}

	var event WebhookEvent
	if err := json.Unmarshal(payload, &event); err != nil {
		return WebhookEvent{}, fmt.Errorf("decode Stripe webhook: %w", err)
	}
	if strings.TrimSpace(event.ID) == "" {
		return WebhookEvent{}, fmt.Errorf("Stripe webhook event ID is required")
	}
	if strings.TrimSpace(event.Type) == "" {
		return WebhookEvent{}, fmt.Errorf("Stripe webhook event type is required")
	}
	return event, nil
}

func (c *Client) doForm(
	ctx context.Context,
	method, path string,
	values url.Values,
	target any,
) error {
	if ctx == nil {
		return fmt.Errorf("Stripe context is required")
	}

	var body io.Reader
	if values != nil {
		body = strings.NewReader(values.Encode())
	}
	req, err := http.NewRequestWithContext(ctx, method, c.baseURL+path, body)
	if err != nil {
		return fmt.Errorf("create Stripe request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+c.secretKey)
	req.Header.Set("Stripe-Version", DefaultAPIVersion)
	req.Header.Set("Accept", "application/json")
	if values != nil {
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("send Stripe request: %w", err)
	}
	defer resp.Body.Close()

	payload, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return fmt.Errorf("read Stripe response: %w", err)
	}
	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return fmt.Errorf("Stripe request failed with status %d: %s", resp.StatusCode, strings.TrimSpace(string(payload)))
	}
	if err := json.Unmarshal(payload, target); err != nil {
		return fmt.Errorf("decode Stripe response: %w", err)
	}
	return nil
}

func parseSignatureHeader(header string) (int64, []string, error) {
	var timestamp int64
	var signatures []string
	for _, part := range strings.Split(header, ",") {
		key, value, ok := strings.Cut(strings.TrimSpace(part), "=")
		if !ok {
			continue
		}
		switch key {
		case "t":
			parsed, err := strconv.ParseInt(value, 10, 64)
			if err != nil {
				return 0, nil, fmt.Errorf("invalid Stripe webhook timestamp")
			}
			timestamp = parsed
		case "v1":
			signatures = append(signatures, value)
		}
	}
	if timestamp == 0 {
		return 0, nil, fmt.Errorf("Stripe webhook timestamp is required")
	}
	if len(signatures) == 0 {
		return 0, nil, fmt.Errorf("Stripe webhook signature is required")
	}
	return timestamp, signatures, nil
}

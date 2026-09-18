package stripe

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"
)

const (
	DefaultBaseURL    = "https://api.stripe.com/v1"
	DefaultAPIVersion = "2026-08-26.dahlia"
)

type Config struct {
	SecretKey     string
	WebhookSecret string
	BaseURL       string
	HTTPClient    *http.Client
}

func DefaultConfig(secretKey, webhookSecret string) Config {
	return Config{
		SecretKey:     strings.TrimSpace(secretKey),
		WebhookSecret: strings.TrimSpace(webhookSecret),
		BaseURL:       DefaultBaseURL,
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("Stripe secret key is required")
	}
	if strings.TrimSpace(c.WebhookSecret) == "" {
		return fmt.Errorf("Stripe webhook secret is required")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("Stripe base URL is required")
	}
	if c.HTTPClient == nil {
		return fmt.Errorf("Stripe HTTP client is required")
	}
	return nil
}

type CreateCheckoutSessionRequest struct {
	AmountMinor int64
	Currency    string
}

func (r CreateCheckoutSessionRequest) Validate() error {
	if r.AmountMinor <= 0 {
		return fmt.Errorf("Stripe checkout amount must be positive")
	}
	if strings.TrimSpace(r.Currency) == "" {
		return fmt.Errorf("Stripe checkout currency is required")
	}
	return nil
}

type CheckoutSession struct {
	ID            string `json:"id"`
	ClientSecret  string `json:"client_secret"`
	Status        string `json:"status"`
	PaymentStatus string `json:"payment_status"`
	Currency      string `json:"currency"`
	AmountTotal   int64  `json:"amount_total"`
}

type WebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object json.RawMessage `json:"object"`
	} `json:"data"`
}

func (e WebhookEvent) DecodeCheckoutSession() (CheckoutSession, error) {
	var session CheckoutSession
	if err := json.Unmarshal(e.Data.Object, &session); err != nil {
		return CheckoutSession{}, fmt.Errorf("decode Stripe checkout session: %w", err)
	}
	if strings.TrimSpace(session.ID) == "" {
		return CheckoutSession{}, fmt.Errorf("Stripe checkout session ID is required")
	}

	return session, nil
}

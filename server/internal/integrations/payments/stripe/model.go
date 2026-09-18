package stripe

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.stripe.com"

type Config struct {
	SecretKey     string
	WebhookSecret string
	BaseURL       string
	APIVersion    string
	HTTPClient    *http.Client
}

func DefaultConfig(secretKey, webhookSecret string) Config {
	return Config{
		SecretKey:     strings.TrimSpace(secretKey),
		WebhookSecret: strings.TrimSpace(webhookSecret),
		BaseURL:       defaultBaseURL,
		HTTPClient:    &http.Client{Timeout: 10 * time.Second},
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("Stripe secret key is required")
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
	Amount      int64
	Currency    string
	Email       string
	Reference   string
	Description string
	ReturnURL   string
	Metadata    map[string]string
}

func (r CreateCheckoutSessionRequest) Validate() error {
	if r.Amount <= 0 {
		return fmt.Errorf("Stripe checkout amount must be positive")
	}
	if strings.TrimSpace(r.Currency) == "" {
		return fmt.Errorf("Stripe checkout currency is required")
	}
	if strings.TrimSpace(r.Email) == "" {
		return fmt.Errorf("Stripe checkout email is required")
	}
	return nil
}

type CheckoutSession struct {
	ID            string            `json:"id"`
	ClientSecret  string            `json:"client_secret"`
	Status        string            `json:"status"`
	PaymentStatus string            `json:"payment_status"`
	PaymentIntent string            `json:"payment_intent"`
	Currency      string            `json:"currency"`
	AmountTotal   int64             `json:"amount_total"`
	Metadata      map[string]string `json:"metadata"`
}

type WebhookEvent struct {
	ID   string `json:"id"`
	Type string `json:"type"`
	Data struct {
		Object CheckoutSession `json:"object"`
	} `json:"data"`
}

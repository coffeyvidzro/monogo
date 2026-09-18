package stripe

import (
	"encoding/json"
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

type Charge struct {
	ID            string            `json:"id"`
	Status        string            `json:"status"`
	Amount        int64             `json:"amount"`
	AmountCaptured int64            `json:"amount_captured"`
	Currency      string            `json:"currency"`
	PaymentIntent string            `json:"payment_intent"`
	Metadata      map[string]string `json:"metadata"`
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

func (e WebhookEvent) DecodeCharge() (Charge, error) {
	var charge Charge
	if err := json.Unmarshal(e.Data.Object, &charge); err != nil {
		return Charge{}, fmt.Errorf("decode Stripe charge: %w", err)
	}
	if strings.TrimSpace(charge.ID) == "" {
		return Charge{}, fmt.Errorf("Stripe charge ID is required")
	}
	return charge, nil
}

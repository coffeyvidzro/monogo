package paystack

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const DefaultBaseURL = "https://api.paystack.co"

type Config struct {
	SecretKey  string
	BaseURL    string
	HTTPClient *http.Client
}

func DefaultConfig(secretKey string) Config {
	return Config{
		SecretKey:  strings.TrimSpace(secretKey),
		BaseURL:    DefaultBaseURL,
		HTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (c Config) Validate() error {
	if strings.TrimSpace(c.SecretKey) == "" {
		return fmt.Errorf("Paystack secret key is required")
	}
	if strings.TrimSpace(c.BaseURL) == "" {
		return fmt.Errorf("Paystack base URL is required")
	}
	if c.HTTPClient == nil {
		return fmt.Errorf("Paystack HTTP client is required")
	}
	return nil
}

type MobileMoneyNetwork string

const (
	MobileMoneyNetworkMTN     MobileMoneyNetwork = "mtn"
	MobileMoneyNetworkAT      MobileMoneyNetwork = "atl"
	MobileMoneyNetworkTelecel MobileMoneyNetwork = "vod"
)

type MobileMoney struct {
	Phone   string             `json:"phone"`
	Network MobileMoneyNetwork `json:"provider"`
}

type ChargeRequest struct {
	Email       string      `json:"email"`
	AmountMinor int64       `json:"amount"`
	MobileMoney MobileMoney `json:"mobile_money"`
}

func (r ChargeRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return fmt.Errorf("Paystack charge email is required")
	}
	if r.AmountMinor <= 0 {
		return fmt.Errorf("Paystack charge amount must be positive")
	}
	if strings.TrimSpace(r.MobileMoney.Phone) == "" {
		return fmt.Errorf("Paystack mobile money phone is required")
	}
	switch r.MobileMoney.Network {
	case MobileMoneyNetworkMTN, MobileMoneyNetworkAT, MobileMoneyNetworkTelecel:
		return nil
	default:
		return fmt.Errorf("unsupported Paystack mobile money network %q", r.MobileMoney.Network)
	}
}

type Charge struct {
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	DisplayText string `json:"display_text"`
}

type Transaction struct {
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	AmountMinor int64  `json:"amount"`
	Currency    string `json:"currency"`
}

type WebhookEvent struct {
	Event string      `json:"event"`
	Data  Transaction `json:"data"`
}

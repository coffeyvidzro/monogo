package paystack

import (
	"fmt"
	"net/http"
	"strings"
	"time"
)

const defaultBaseURL = "https://api.paystack.co"

type Config struct {
	SecretKey  string
	BaseURL    string
	HTTPClient *http.Client
}

func DefaultConfig(secretKey string) Config {
	return Config{
		SecretKey:  strings.TrimSpace(secretKey),
		BaseURL:    defaultBaseURL,
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

type MobileMoneyProvider string

const (
	MobileMoneyMTN     MobileMoneyProvider = "mtn"
	MobileMoneyATMoney MobileMoneyProvider = "atl"
	MobileMoneyTelecel MobileMoneyProvider = "vod"
)

type MobileMoney struct {
	Phone    string              `json:"phone"`
	Provider MobileMoneyProvider `json:"provider"`
}

type ChargeRequest struct {
	Email       string       `json:"email"`
	Amount      int64        `json:"amount"`
	MobileMoney *MobileMoney `json:"mobile_money"`
}

func (r ChargeRequest) Validate() error {
	if strings.TrimSpace(r.Email) == "" {
		return fmt.Errorf("Paystack charge email is required")
	}
	if r.Amount <= 0 {
		return fmt.Errorf("Paystack charge amount must be positive")
	}
	if r.MobileMoney == nil {
		return fmt.Errorf("Paystack mobile money details are required")
	}
	if strings.TrimSpace(r.MobileMoney.Phone) == "" {
		return fmt.Errorf("Paystack mobile money phone is required")
	}
	switch r.MobileMoney.Provider {
	case MobileMoneyMTN, MobileMoneyATMoney, MobileMoneyTelecel:
	default:
		return fmt.Errorf("unsupported Paystack mobile money provider %q", r.MobileMoney.Provider)
	}
	return nil
}

type ChargeData struct {
	Reference   string `json:"reference"`
	Status      string `json:"status"`
	DisplayText string `json:"display_text"`
}

type ChargeResponse struct {
	Status  bool       `json:"status"`
	Message string     `json:"message"`
	Data    ChargeData `json:"data"`
}

type TransactionData struct {
	Reference string `json:"reference"`
	Status    string `json:"status"`
	Message   string `json:"message"`
	Amount    int64  `json:"amount"`
	Currency  string `json:"currency"`
}

type TransactionResponse struct {
	Status  bool            `json:"status"`
	Message string          `json:"message"`
	Data    TransactionData `json:"data"`
}

type WebhookEvent struct {
	Event string          `json:"event"`
	Data  TransactionData `json:"data"`
}

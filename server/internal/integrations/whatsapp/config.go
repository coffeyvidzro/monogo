package whatsapp

import (
	"fmt"
	"net/url"
	"strings"
	"time"
)

type Config struct {
	BaseURL, AccessToken, PhoneNumberID, AppSecret string
	Timeout                                        time.Duration
}

func (c Config) Validate() error {
	u, err := url.Parse(c.BaseURL)
	if err != nil || u.Scheme != "https" || u.Host == "" {
		return fmt.Errorf("WhatsApp base URL must be absolute HTTPS")
	}
	if strings.TrimSpace(c.AccessToken) == "" || strings.TrimSpace(c.PhoneNumberID) == "" {
		return fmt.Errorf("WhatsApp access token and phone number id are required")
	}
	return nil
}

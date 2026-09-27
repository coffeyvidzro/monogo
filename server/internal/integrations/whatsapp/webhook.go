package whatsapp

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"strings"
)

// VerifySignature authenticates X-Hub-Signature-256 before webhook parsing.
func (c *Client) VerifySignature(body []byte, signature string) bool {
	if c == nil || c.config.AppSecret == "" {
		return false
	}
	provided, err := hex.DecodeString(strings.TrimPrefix(signature, "sha256="))
	if err != nil {
		return false
	}
	mac := hmac.New(sha256.New, []byte(c.config.AppSecret))
	_, _ = mac.Write(body)
	return hmac.Equal(provided, mac.Sum(nil))
}

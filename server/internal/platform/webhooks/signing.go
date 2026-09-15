package webhooks

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"
)

const signatureHeader = "X-Leamout-Signature"

func newSigningSecret() ([]byte, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return nil, fmt.Errorf("generate signing secret: %w", err)
	}
	return b, nil
}
func sign(secret, body []byte, timestamp time.Time) string {
	payload := fmt.Sprintf("%d.%s", timestamp.Unix(), body)
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(payload))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}
func encodeSecret(secret []byte) string { return base64.RawURLEncoding.EncodeToString(secret) }

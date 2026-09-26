package tools

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"time"

	"github.com/google/uuid"
)

const toolSignatureHeader = "X-Leamout-Signature"

func newSigningSecret() (string, error) {
	raw := make([]byte, 32)
	if _, err := rand.Read(raw); err != nil {
		return "", fmt.Errorf("generate Voice Agent tool signing secret: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(raw), nil
}

func signingScope(organizationID, agentID, toolID uuid.UUID) string {
	return "voice-agent-tool-signing:" +
		organizationID.String() + ":" +
		agentID.String() + ":" +
		toolID.String()
}

func signToolRequest(secret string, body []byte, timestamp time.Time) string {
	payload := fmt.Sprintf("%d.%s", timestamp.Unix(), body)
	mac := hmac.New(sha256.New, []byte(secret))
	_, _ = mac.Write([]byte(payload))
	return "v1=" + hex.EncodeToString(mac.Sum(nil))
}

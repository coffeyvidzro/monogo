package coturn

import (
	"crypto/hmac"
	"crypto/sha1" // #nosec G505 -- Coturn TURN REST authentication requires HMAC-SHA1.
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"time"

	"github.com/google/uuid"
)

// Credential is compatible with Coturn's use-auth-secret mechanism.
type Credential struct {
	Username  string
	Password  string
	ExpiresAt time.Time
}

func (c *Client) issue(organizationID uuid.UUID, now time.Time, nonceSource io.Reader) (Credential, error) {
	nonce := make([]byte, 16)
	if _, err := io.ReadFull(nonceSource, nonce); err != nil {
		return Credential{}, fmt.Errorf("generate TURN credential nonce: %w", err)
	}

	expiresAt := now.UTC().Add(10 * time.Minute)
	username := fmt.Sprintf("%d:%s:%s", expiresAt.Unix(), organizationID, hex.EncodeToString(nonce))
	digest := hmac.New(sha1.New, []byte(c.authSecret))
	_, _ = digest.Write([]byte(username))
	return Credential{
		Username:  username,
		Password:  base64.StdEncoding.EncodeToString(digest.Sum(nil)),
		ExpiresAt: expiresAt,
	}, nil
}

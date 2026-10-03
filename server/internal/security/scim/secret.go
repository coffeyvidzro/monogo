package scim

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
)

const tokenPrefix = "lm_scim_"

func generateToken() (string, string, string, error) {
	secret := make([]byte, 32)
	if _, err := rand.Read(secret); err != nil {
		return "", "", "", fmt.Errorf("generate SCIM token: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(secret)
	prefix := tokenPrefix + encoded[:8]
	token := prefix + "_" + encoded
	return token, prefix, hashToken(token), nil
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

package token

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
)

// Generate returns sizeBytes of cryptographically secure random material as a
// hexadecimal token. Hex encoding preserves the format used by existing
// Leamout session tokens.
func Generate(sizeBytes int) (string, error) {
	if sizeBytes <= 0 {
		return "", fmt.Errorf("token size must be positive")
	}

	value := make([]byte, sizeBytes)
	if _, err := rand.Read(value); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}

	return hex.EncodeToString(value), nil
}

// Hash returns the SHA-256 hex digest used for persisted lookup values. A fast
// hash is appropriate for high-entropy generated tokens and one-time codes.
func Hash(value string) string {
	sum := sha256.Sum256([]byte(value))
	return hex.EncodeToString(sum[:])
}

// Verify compares a presented value with a persisted digest.
func Verify(value, expectedHash string) bool {
	actual := Hash(value)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}

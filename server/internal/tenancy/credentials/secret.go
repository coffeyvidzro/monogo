package credentials

import (
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/base64"
	"errors"
	"fmt"
)

const (
	tokenPrefix = "lm_org_"
	secretBytes = 32
)

var ErrInvalidToken = errors.New("invalid organization token")

// GenerateToken creates a new opaque organization credential and its lookup
// values. The returned token is the only copy of the usable secret.
func GenerateToken() (token string, prefix string, hash string, err error) {
	secret := make([]byte, secretBytes)
	if _, err := rand.Read(secret); err != nil {
		return "", "", "", fmt.Errorf("generate organization token: %w", err)
	}

	encoded := base64.RawURLEncoding.EncodeToString(secret)
	prefix = tokenPrefix + encoded[:8]
	token = prefix + "_" + encoded
	hash = HashToken(token)

	return token, prefix, hash, nil
}

// HashToken returns the SHA-256 digest used to persist an organization token.
// The token is already generated from high-entropy random material, so a fast
// cryptographic hash is appropriate for lookup and verification.
func HashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return base64.RawURLEncoding.EncodeToString(sum[:])
}

// VerifyToken compares a presented token with a persisted token hash.
func VerifyToken(token, expectedHash string) bool {
	actual := HashToken(token)
	return subtle.ConstantTimeCompare([]byte(actual), []byte(expectedHash)) == 1
}

// ParseTokenPrefix returns the non-secret lookup prefix from a generated token.
func ParseTokenPrefix(token string) (string, error) {
	if len(token) < len(tokenPrefix)+8+1 || token[:len(tokenPrefix)] != tokenPrefix {
		return "", ErrInvalidToken
	}

	separator := len(tokenPrefix) + 8
	if token[separator] != '_' || len(token) == separator+1 {
		return "", ErrInvalidToken
	}

	return token[:separator], nil
}

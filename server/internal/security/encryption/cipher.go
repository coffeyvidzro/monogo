// Package encryption provides authenticated encryption for persisted secrets.
package encryption

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"io"
	"strings"
)

type Cipher struct{ aead cipher.AEAD }

const scopedEnvelopeVersion = "v1"

func New(encodedKey string) (*Cipher, error) {
	key, err := base64.RawURLEncoding.DecodeString(encodedKey)
	if err != nil {
		return nil, fmt.Errorf("decode credential encryption key: %w", err)
	}
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("initialize credential encryption: %w", err)
	}
	aead, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("initialize credential encryption mode: %w", err)
	}
	return &Cipher{aead: aead}, nil
}

func (c *Cipher) Encrypt(plaintext string) (string, error) {
	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate credential nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), nil)
	return base64.RawURLEncoding.EncodeToString(sealed), nil
}

func (c *Cipher) Decrypt(encoded string) (string, error) {
	return c.decrypt(encoded, nil)
}

// EncryptForScope encrypts a secret and cryptographically binds it to its
// owner and purpose. Callers should construct scope from stable, non-secret
// identifiers (for example, organization, resource, and credential direction).
// A ciphertext copied to another tenant or field will then fail authentication.
func (c *Cipher) EncryptForScope(scope, plaintext string) (string, error) {
	associatedData, err := scopedAssociatedData(scope)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, c.aead.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", fmt.Errorf("generate credential nonce: %w", err)
	}
	sealed := c.aead.Seal(nonce, nonce, []byte(plaintext), associatedData)
	return scopedEnvelopeVersion + "." + base64.RawURLEncoding.EncodeToString(sealed), nil
}

// DecryptForScope decrypts a scoped envelope only when the exact scope used at
// encryption is supplied. It intentionally rejects legacy, unscoped values so
// security-sensitive callers cannot silently lose tenant binding.
func (c *Cipher) DecryptForScope(scope, encoded string) (string, error) {
	associatedData, err := scopedAssociatedData(scope)
	if err != nil {
		return "", err
	}
	version, payload, ok := strings.Cut(encoded, ".")
	if !ok || version != scopedEnvelopeVersion {
		return "", fmt.Errorf("unsupported scoped credential envelope")
	}
	return c.decrypt(payload, associatedData)
}

func (c *Cipher) decrypt(encoded string, associatedData []byte) (string, error) {
	sealed, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return "", fmt.Errorf("decode encrypted credential: %w", err)
	}
	if len(sealed) < c.aead.NonceSize() {
		return "", fmt.Errorf("encrypted credential is truncated")
	}
	nonce, ciphertext := sealed[:c.aead.NonceSize()], sealed[c.aead.NonceSize():]
	plaintext, err := c.aead.Open(nil, nonce, ciphertext, associatedData)
	if err != nil {
		return "", fmt.Errorf("decrypt credential: %w", err)
	}
	return string(plaintext), nil
}

func scopedAssociatedData(scope string) ([]byte, error) {
	scope = strings.TrimSpace(scope)
	if scope == "" {
		return nil, fmt.Errorf("credential encryption scope is required")
	}
	return []byte("leamout:credential:" + scopedEnvelopeVersion + ":" + scope), nil
}

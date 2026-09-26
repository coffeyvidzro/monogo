package transport

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

const minimumTokenSecretBytes = 32

type TokenClaims struct {
	SessionID      uuid.UUID `json:"sid"`
	CallID         uuid.UUID `json:"cid"`
	ChannelID      uuid.UUID `json:"chid"`
	OrganizationID uuid.UUID `json:"oid"`
	ExpiresAt      int64     `json:"exp"`
	Nonce          string    `json:"nonce"`
}

type TokenService struct {
	secret []byte
	now    func() time.Time
	random func([]byte) (int, error)

	mu   sync.Mutex
	used map[string]int64
}

func NewTokenService(secret string) (*TokenService, error) {
	secret = strings.TrimSpace(secret)
	if len(secret) < minimumTokenSecretBytes {
		return nil, fmt.Errorf("media token secret must contain at least %d bytes", minimumTokenSecretBytes)
	}
	return &TokenService{
		secret: []byte(secret),
		now:    time.Now,
		random: rand.Read,
		used:   make(map[string]int64),
	}, nil
}

func (s *TokenService) Issue(claims TokenClaims, ttl time.Duration) (string, error) {
	if err := validateTokenIdentity(claims); err != nil {
		return "", err
	}
	if ttl <= 0 {
		return "", fmt.Errorf("media token TTL must be positive")
	}

	nonce := make([]byte, 16)
	if _, err := s.random(nonce); err != nil {
		return "", fmt.Errorf("generate media token nonce: %w", err)
	}
	claims.Nonce = hex.EncodeToString(nonce)
	claims.ExpiresAt = s.now().UTC().Add(ttl).Unix()

	payload, err := json.Marshal(claims)
	if err != nil {
		return "", fmt.Errorf("marshal media token: %w", err)
	}
	encoded := base64.RawURLEncoding.EncodeToString(payload)
	return encoded + "." + s.signature(encoded), nil
}

func (s *TokenService) VerifyAndConsume(token string) (TokenClaims, error) {
	encoded, signature, ok := strings.Cut(strings.TrimSpace(token), ".")
	if !ok || encoded == "" || signature == "" {
		return TokenClaims{}, fmt.Errorf("invalid media token format")
	}
	if !hmac.Equal([]byte(signature), []byte(s.signature(encoded))) {
		return TokenClaims{}, fmt.Errorf("invalid media token signature")
	}
	payload, err := base64.RawURLEncoding.DecodeString(encoded)
	if err != nil {
		return TokenClaims{}, fmt.Errorf("decode media token: %w", err)
	}
	var claims TokenClaims
	if err := json.Unmarshal(payload, &claims); err != nil {
		return TokenClaims{}, fmt.Errorf("unmarshal media token: %w", err)
	}
	if err := validateTokenIdentity(claims); err != nil {
		return TokenClaims{}, err
	}
	if claims.Nonce == "" {
		return TokenClaims{}, fmt.Errorf("media token nonce is required")
	}

	now := s.now().UTC().Unix()
	if claims.ExpiresAt <= now {
		return TokenClaims{}, fmt.Errorf("media token expired")
	}

	s.mu.Lock()
	defer s.mu.Unlock()
	for nonce, expiresAt := range s.used {
		if expiresAt <= now {
			delete(s.used, nonce)
		}
	}
	if _, exists := s.used[claims.Nonce]; exists {
		return TokenClaims{}, fmt.Errorf("media token already used")
	}
	s.used[claims.Nonce] = claims.ExpiresAt
	return claims, nil
}

func (s *TokenService) signature(payload string) string {
	digest := hmac.New(sha256.New, s.secret)
	_, _ = digest.Write([]byte(payload))
	return base64.RawURLEncoding.EncodeToString(digest.Sum(nil))
}

func validateTokenIdentity(claims TokenClaims) error {
	if claims.SessionID == uuid.Nil {
		return fmt.Errorf("media token session id is required")
	}
	if claims.CallID == uuid.Nil {
		return fmt.Errorf("media token call id is required")
	}
	if claims.ChannelID == uuid.Nil {
		return fmt.Errorf("media token channel id is required")
	}
	if claims.OrganizationID == uuid.Nil {
		return fmt.Errorf("media token organization id is required")
	}
	return nil
}

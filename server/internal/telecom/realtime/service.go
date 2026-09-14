package realtime

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha1" // #nosec G505 -- coturn's TURN REST authentication protocol requires HMAC-SHA1.
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strings"
	"time"

	"github.com/google/uuid"
)

// ErrIssueRateLimited indicates that an organization exhausted its shared
// credential-issuance quota.
var ErrIssueRateLimited = errors.New("TURN credential issuance rate limit exceeded")

// IssueLimiter coordinates credential quotas across API replicas.
type IssueLimiter interface {
	AllowFixedWindow(context.Context, string, int64, time.Duration) (bool, error)
}

type Service struct {
	config  Config
	now     func() time.Time
	random  io.Reader
	limiter IssueLimiter
}

func NewService(config Config, limiter IssueLimiter) (*Service, error) {
	for index := range config.URLs {
		config.URLs[index] = strings.TrimSpace(config.URLs[index])
	}
	if err := validateConfig(config); err != nil {
		return nil, err
	}
	if limiter == nil {
		return nil, fmt.Errorf("TURN credential issue limiter is required")
	}
	return &Service{config: config, now: time.Now, random: rand.Reader, limiter: limiter}, nil
}

// Issue creates short-lived credentials compatible with coturn's
// use-auth-secret mechanism. The expiry is encoded as the first username field
// so coturn can reject the credential without calling the Leamout API.
func (s *Service) Issue(ctx context.Context, organizationID uuid.UUID) (ICECredentials, error) {
	if ctx == nil {
		return ICECredentials{}, fmt.Errorf("context is required")
	}
	if organizationID == uuid.Nil {
		return ICECredentials{}, fmt.Errorf("organization id is required")
	}
	allowed, err := s.limiter.AllowFixedWindow(
		ctx,
		"ratelimit:turn-credentials:"+organizationID.String(),
		60,
		time.Minute,
	)
	if err != nil {
		return ICECredentials{}, fmt.Errorf("rate limit TURN credential issuance: %w", err)
	}
	if !allowed {
		return ICECredentials{}, ErrIssueRateLimited
	}

	nonce := make([]byte, 16)
	if _, err := io.ReadFull(s.random, nonce); err != nil {
		return ICECredentials{}, fmt.Errorf("generate TURN credential nonce: %w", err)
	}

	expiresAt := s.now().UTC().Add(10 * time.Minute)
	username := fmt.Sprintf("%d:%s:%s", expiresAt.Unix(), organizationID, hex.EncodeToString(nonce))
	digest := hmac.New(sha1.New, []byte(s.config.AuthSecret))
	_, _ = digest.Write([]byte(username))
	credential := base64.StdEncoding.EncodeToString(digest.Sum(nil))

	return ICECredentials{
		ICEServers: []ICEServer{{
			URLs:       append([]string(nil), s.config.URLs...),
			Username:   username,
			Credential: credential,
		}},
		ExpiresAt: expiresAt,
	}, nil
}

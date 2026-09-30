package realtime

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"time"

	"github.com/coffeyvidzro/monogo/internal/integrations/coturn"
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
	coturn  *coturn.Client
	now     func() time.Time
	random  io.Reader
	limiter IssueLimiter
}

func NewService(coturnClient *coturn.Client, limiter IssueLimiter) (*Service, error) {
	if coturnClient == nil {
		return nil, fmt.Errorf("coturn client is required")
	}
	if limiter == nil {
		return nil, fmt.Errorf("TURN credential issue limiter is required")
	}
	return &Service{coturn: coturnClient, now: time.Now, random: rand.Reader, limiter: limiter}, nil
}

// Issue applies organization-scoped rate limiting before delegating TURN
// credential generation to the Coturn integration.
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

	issued, err := s.coturn.Issue(organizationID, s.now(), s.random)
	if err != nil {
		return ICECredentials{}, fmt.Errorf("issue TURN credential: %w", err)
	}
	return ICECredentials{
		ICEServers: []ICEServer{{
			URLs:       s.coturn.URLs(),
			Username:   issued.Username,
			Credential: issued.Password,
		}},
		ExpiresAt: issued.ExpiresAt,
	}, nil
}

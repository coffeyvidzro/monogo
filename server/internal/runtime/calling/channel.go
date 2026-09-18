package calling

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

var (
	ErrChannelUnavailable  = errors.New("active call channel unavailable")
	ErrAdmissionCPS        = errors.New("carrier CPS limit exceeded")
	ErrAdmissionConcurrent = errors.New("carrier concurrent call limit exceeded")
)

const callAdmissionLeaseTTL = 26 * time.Hour

type ChannelStore struct {
	client *redisintegration.Client
}

func NewChannelStore(client *redisintegration.Client) *ChannelStore {
	if client == nil {
		panic("calling: Redis client is required")
	}
	return &ChannelStore{client: client}
}

func (s *ChannelStore) Bind(ctx context.Context, callID uuid.UUID, channelID string) error {
	channelID = strings.TrimSpace(channelID)
	if callID == uuid.Nil || channelID == "" {
		return fmt.Errorf("call id and channel id are required")
	}
	return s.client.Set(ctx, channelKey(callID), channelID, 24*time.Hour)
}

func (s *ChannelStore) Get(ctx context.Context, callID uuid.UUID) (string, error) {
	if callID == uuid.Nil {
		return "", fmt.Errorf("call id is required")
	}

	channelID, err := s.client.Get(ctx, channelKey(callID))
	if errors.Is(err, redisv9.Nil) {
		return "", ErrChannelUnavailable
	}
	if err != nil {
		return "", err
	}

	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", ErrChannelUnavailable
	}
	return channelID, nil
}

func (s *ChannelStore) Delete(ctx context.Context, callID uuid.UUID) error {
	if callID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}
	return s.client.Delete(ctx, channelKey(callID))
}

type AdmissionLimiter struct {
	client *redisintegration.Client
}

func NewAdmissionLimiter(client *redisintegration.Client) *AdmissionLimiter {
	if client == nil {
		panic("calling: Redis admission store is required")
	}
	return &AdmissionLimiter{client: client}
}

func (l *AdmissionLimiter) Acquire(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	leaseID string,
	limits routing.Limits,
) error {
	if carrierConnectionID == uuid.Nil || leaseID == "" {
		return fmt.Errorf("carrier connection id and lease id are required")
	}
	if limits.MaxCPS < 1 || limits.MaxConcurrentCalls < 1 {
		return fmt.Errorf("carrier admission limits must be positive")
	}

	allowed, reason, err := l.client.AcquireCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		leaseID,
		int64(limits.MaxCPS),
		int64(limits.MaxConcurrentCalls),
		callAdmissionLeaseTTL,
	)
	if err != nil {
		return fmt.Errorf("acquire carrier call lease: %w", err)
	}
	if allowed {
		return nil
	}

	switch reason {
	case "cps":
		return ErrAdmissionCPS
	case "concurrent":
		return ErrAdmissionConcurrent
	default:
		return fmt.Errorf("carrier admission rejected: %s", reason)
	}
}

func (l *AdmissionLimiter) Bind(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	leaseID string,
	callID uuid.UUID,
) error {
	if carrierConnectionID == uuid.Nil || leaseID == "" || callID == uuid.Nil {
		return fmt.Errorf("carrier connection id, lease id, and call id are required")
	}
	return l.client.BindCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		leaseID,
		callID.String(),
	)
}

func (l *AdmissionLimiter) Release(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	callOrLeaseID string,
) error {
	if carrierConnectionID == uuid.Nil || callOrLeaseID == "" {
		return fmt.Errorf("carrier connection id and call or lease id are required")
	}
	return l.client.ReleaseCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		callOrLeaseID,
	)
}

func (l *AdmissionLimiter) Refresh(
	ctx context.Context,
	carrierConnectionID, callID uuid.UUID,
) error {
	if carrierConnectionID == uuid.Nil || callID == uuid.Nil {
		return fmt.Errorf("carrier connection id and call id are required")
	}
	return l.client.RefreshCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		callID.String(),
		callAdmissionLeaseTTL,
	)
}

func channelKey(callID uuid.UUID) string {
	return "telecom:calls:channel:" + callID.String()
}

func admissionPrefix(carrierConnectionID uuid.UUID) string {
	return "telecom:admission:carrier:" + carrierConnectionID.String()
}

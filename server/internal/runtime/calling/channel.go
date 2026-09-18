package calling

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/telecom/calls"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
	redisv9 "github.com/redis/go-redis/v9"
)

const callAdmissionLeaseTTL = 26 * time.Hour

type RedisChannelStore struct {
	client *redisintegration.Client
}

var _ calls.ChannelStore = (*RedisChannelStore)(nil)

func NewRedisChannelStore(client *redisintegration.Client) *RedisChannelStore {
	if client == nil {
		panic("calling: Redis client is required")
	}
	return &RedisChannelStore{client: client}
}

func (s *RedisChannelStore) Bind(ctx context.Context, callID uuid.UUID, channelID string) error {
	channelID = strings.TrimSpace(channelID)
	if callID == uuid.Nil || channelID == "" {
		return fmt.Errorf("call id and channel id are required")
	}
	return s.client.Set(ctx, channelKey(callID), channelID, 24*time.Hour)
}

func (s *RedisChannelStore) Get(ctx context.Context, callID uuid.UUID) (string, error) {
	if callID == uuid.Nil {
		return "", fmt.Errorf("call id is required")
	}
	channelID, err := s.client.Get(ctx, channelKey(callID))
	if errors.Is(err, redisv9.Nil) {
		return "", calls.ErrChannelUnavailable
	}
	if err != nil {
		return "", err
	}
	channelID = strings.TrimSpace(channelID)
	if channelID == "" {
		return "", calls.ErrChannelUnavailable
	}
	return channelID, nil
}

func (s *RedisChannelStore) Delete(ctx context.Context, callID uuid.UUID) error {
	if callID == uuid.Nil {
		return fmt.Errorf("call id is required")
	}
	return s.client.Delete(ctx, channelKey(callID))
}

type callLeaseStore interface {
	AcquireCallLease(context.Context, string, string, int64, int64, time.Duration) (bool, string, error)
	BindCallLease(context.Context, string, string, string) error
	ReleaseCallLease(context.Context, string, string) error
	RefreshCallLease(context.Context, string, string, time.Duration) error
}

type dailyUsageReader interface {
	CarrierDailyUsageSeconds(context.Context, uuid.UUID) (int64, error)
}

type RedisAdmissionLimiter struct {
	store callLeaseStore
	usage dailyUsageReader
}

var _ calls.AdmissionLimiter = (*RedisAdmissionLimiter)(nil)

func NewRedisAdmissionLimiter(
	client *redisintegration.Client,
	usage dailyUsageReader,
) *RedisAdmissionLimiter {
	if client == nil {
		panic("calling: Redis admission store is required")
	}
	if usage == nil {
		panic("calling: daily usage reader is required")
	}
	return &RedisAdmissionLimiter{store: client, usage: usage}
}

func (l *RedisAdmissionLimiter) Acquire(
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

	if limits.MaxDailyMinutes != nil {
		usedSeconds, err := l.usage.CarrierDailyUsageSeconds(ctx, carrierConnectionID)
		if err != nil {
			return fmt.Errorf("read carrier daily usage: %w", err)
		}
		if usedSeconds >= *limits.MaxDailyMinutes*60 {
			return calls.ErrAdmissionDailyMinutes
		}
	}

	allowed, reason, err := l.store.AcquireCallLease(
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
		return calls.ErrAdmissionCPS
	case "concurrent":
		return calls.ErrAdmissionConcurrent
	default:
		return fmt.Errorf("carrier admission rejected: %s", reason)
	}
}

func (l *RedisAdmissionLimiter) Bind(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	leaseID string,
	callID uuid.UUID,
) error {
	if carrierConnectionID == uuid.Nil || leaseID == "" || callID == uuid.Nil {
		return fmt.Errorf("carrier connection id, lease id, and call id are required")
	}
	return l.store.BindCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		leaseID,
		callID.String(),
	)
}

func (l *RedisAdmissionLimiter) Release(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	callOrLeaseID string,
) error {
	if carrierConnectionID == uuid.Nil || callOrLeaseID == "" {
		return fmt.Errorf("carrier connection id and call or lease id are required")
	}
	return l.store.ReleaseCallLease(
		ctx,
		admissionPrefix(carrierConnectionID),
		callOrLeaseID,
	)
}

func (l *RedisAdmissionLimiter) Refresh(
	ctx context.Context,
	carrierConnectionID, callID uuid.UUID,
) error {
	if carrierConnectionID == uuid.Nil || callID == uuid.Nil {
		return fmt.Errorf("carrier connection id and call id are required")
	}
	return l.store.RefreshCallLease(
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

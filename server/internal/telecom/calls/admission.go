package calls

import (
	"context"
	"errors"
	"fmt"
	"time"

	redisintegration "github.com/coffeyvidzro/monogo/internal/integrations/redis"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
)

var (
	ErrAdmissionCPS             = errors.New("carrier CPS limit exceeded")
	ErrAdmissionConcurrent      = errors.New("carrier concurrent call limit exceeded")
	ErrAdmissionDailyMinutes    = errors.New("carrier daily minute limit exceeded")
)

const callAdmissionLeaseTTL = 26 * time.Hour

type AdmissionLimiter interface {
	Acquire(context.Context, uuid.UUID, string, routing.Limits) error
	Bind(context.Context, uuid.UUID, string, uuid.UUID) error
	Release(context.Context, uuid.UUID, string) error
	Refresh(context.Context, uuid.UUID, uuid.UUID) error
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

func NewRedisAdmissionLimiter(
	client *redisintegration.Client,
	usage dailyUsageReader,
) *RedisAdmissionLimiter {
	if client == nil {
		panic("calls: Redis admission store is required")
	}
	if usage == nil {
		panic("calls: daily usage reader is required")
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
			return ErrAdmissionDailyMinutes
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
		return ErrAdmissionCPS
	case "concurrent":
		return ErrAdmissionConcurrent
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

func admissionPrefix(carrierConnectionID uuid.UUID) string {
	return "telecom:admission:carrier:" + carrierConnectionID.String()
}

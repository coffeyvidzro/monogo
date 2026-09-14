package calls

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
)

var (
	ErrCPSLimit        = errors.New("carrier calls-per-second limit reached")
	ErrConcurrentLimit = errors.New("carrier concurrent-call limit reached")
	ErrDailyLimit      = errors.New("carrier daily-minute limit reached")
)

type CallLimits struct {
	CarrierConnectionID uuid.UUID
	MaxCPS              int32
	MaxConcurrent       int32
	MaxDailyMinutes     *int64
}

type leaseCoordinator interface {
	AcquireCallLease(context.Context, string, string, int64, int64, time.Duration) (bool, string, error)
	BindCallLease(context.Context, string, string, string) error
	ReleaseCallLease(context.Context, string, string) error
	RefreshCallLease(context.Context, string, string, time.Duration) error
}

type dailyUsageStore interface {
	CarrierDailySeconds(context.Context, uuid.UUID, time.Time) (int64, error)
}

type AdmissionController struct {
	leases leaseCoordinator
	usage  dailyUsageStore
	now    func() time.Time
}

func NewAdmissionController(leases leaseCoordinator, usage dailyUsageStore) (*AdmissionController, error) {
	if leases == nil || usage == nil {
		return nil, fmt.Errorf("call admission lease coordinator and usage store are required")
	}
	return &AdmissionController{leases: leases, usage: usage, now: time.Now}, nil
}

func (a *AdmissionController) Admit(ctx context.Context, limits CallLimits) (string, error) {
	if limits.CarrierConnectionID == uuid.Nil || limits.MaxCPS <= 0 || limits.MaxConcurrent <= 0 {
		return "", fmt.Errorf("valid carrier call limits are required")
	}
	if limits.MaxDailyMinutes != nil {
		now := a.now().UTC()
		day := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, time.UTC)
		seconds, err := a.usage.CarrierDailySeconds(ctx, limits.CarrierConnectionID, day)
		if err != nil {
			return "", fmt.Errorf("calculate durable carrier daily usage: %w", err)
		}
		if seconds >= *limits.MaxDailyMinutes*60 {
			return "", ErrDailyLimit
		}
	}
	leaseID := uuid.NewString()
	prefix := "call-quota:" + limits.CarrierConnectionID.String()
	allowed, reason, err := a.leases.AcquireCallLease(ctx, prefix, leaseID, int64(limits.MaxCPS), int64(limits.MaxConcurrent), 6*time.Hour)
	if err != nil {
		return "", fmt.Errorf("coordinate carrier call admission: %w", err)
	}
	if !allowed {
		if reason == "cps" {
			return "", ErrCPSLimit
		}
		return "", ErrConcurrentLimit
	}
	return leaseID, nil
}

func (a *AdmissionController) Bind(ctx context.Context, carrierID uuid.UUID, leaseID, callID string) error {
	return a.leases.BindCallLease(ctx, "call-quota:"+carrierID.String(), leaseID, callID)
}

func (a *AdmissionController) Release(ctx context.Context, carrierID uuid.UUID, callOrLeaseID string) error {
	if carrierID == uuid.Nil || callOrLeaseID == "" {
		return nil
	}
	return a.leases.ReleaseCallLease(ctx, "call-quota:"+carrierID.String(), callOrLeaseID)
}

func (a *AdmissionController) Refresh(ctx context.Context, carrierID uuid.UUID, callID string) error {
	if carrierID == uuid.Nil || callID == "" {
		return nil
	}
	return a.leases.RefreshCallLease(ctx, "call-quota:"+carrierID.String(), callID, 6*time.Hour)
}

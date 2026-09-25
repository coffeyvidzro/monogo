package payments

import (
	"context"
	"fmt"
	"time"
)

type recoveryRepository interface {
	ListRecovery(context.Context, int32) ([]RecoveryCandidate, error)
}

type settlementService interface {
	Settle(context.Context, VerifiedSettlement) (SettlementResult, error)
}

type ProviderVerifier interface {
	Verify(context.Context, RecoveryCandidate) (Verification, error)
}

type RecoveryJob struct {
	repository recoveryRepository
	service    settlementService
	verifiers  map[string]ProviderVerifier
	batch      int32
	interval   time.Duration
}

func NewRecoveryJob(repository recoveryRepository, service settlementService, verifiers map[string]ProviderVerifier, batch int32, interval time.Duration) (*RecoveryJob, error) {
	if repository == nil || service == nil {
		return nil, fmt.Errorf("payment recovery dependencies are required")
	}
	if batch < 1 || batch > 500 {
		return nil, fmt.Errorf("payment recovery batch must be between 1 and 500")
	}
	if interval <= 0 {
		return nil, fmt.Errorf("payment recovery interval must be positive")
	}
	return &RecoveryJob{repository: repository, service: service, verifiers: verifiers, batch: batch, interval: interval}, nil
}

func (j *RecoveryJob) RunOnce(ctx context.Context) error {
	candidates, err := j.repository.ListRecovery(ctx, j.batch)
	if err != nil {
		return fmt.Errorf("list payments due for recovery: %w", err)
	}
	for _, candidate := range candidates {
		verifier := j.verifiers[candidate.Provider]
		if verifier == nil {
			continue
		}
		verification, err := verifier.Verify(ctx, candidate)
		if err != nil {
			return fmt.Errorf("verify payment %s: %w", candidate.PaymentID, err)
		}
		if !verification.Succeeded {
			continue
		}
		_, err = j.service.Settle(ctx, VerifiedSettlement{OrganizationID: candidate.OrganizationID,
			PaymentID: candidate.PaymentID, Provider: candidate.Provider, ProviderReference: candidate.ProviderReference,
			AmountMinor: verification.AmountMinor, Currency: verification.Currency, VerifiedAt: verification.VerifiedAt})
		if err != nil {
			return fmt.Errorf("settle recovered payment %s: %w", candidate.PaymentID, err)
		}
	}
	return nil
}

func (j *RecoveryJob) Run(ctx context.Context) error {
	ticker := time.NewTicker(j.interval)
	defer ticker.Stop()
	for {
		if err := j.RunOnce(ctx); err != nil {
			return err
		}
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
		}
	}
}

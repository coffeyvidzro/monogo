package payments

import (
	"context"
	"testing"
	"time"

	"github.com/google/uuid"
)

type recoveryRepoStub struct{ candidates []RecoveryCandidate }

func (s recoveryRepoStub) ListRecovery(context.Context, int32) ([]RecoveryCandidate, error) {
	return s.candidates, nil
}

type settlementStub struct{ requests []VerifiedSettlement }

func (s *settlementStub) Settle(_ context.Context, request VerifiedSettlement) (SettlementResult, error) {
	s.requests = append(s.requests, request)
	return SettlementResult{}, nil
}

type verifierStub struct{ verification Verification }

func (s verifierStub) Verify(context.Context, RecoveryCandidate) (Verification, error) {
	return s.verification, nil
}

func TestRecoveryJobSettlesVerifiedSuccess(t *testing.T) {
	candidate := RecoveryCandidate{OrganizationID: uuid.New(), PaymentID: uuid.New(), Provider: "stripe",
		ProviderReference: "cs_123", AmountMinor: 500, Currency: "USD"}
	service := &settlementStub{}
	job, err := NewRecoveryJob(recoveryRepoStub{candidates: []RecoveryCandidate{candidate}}, service,
		map[string]ProviderVerifier{"stripe": verifierStub{verification: Verification{Succeeded: true,
			AmountMinor: 500, Currency: "USD", VerifiedAt: time.Now()}}}, 10, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(service.requests) != 1 || service.requests[0].PaymentID != candidate.PaymentID {
		t.Fatalf("requests = %+v", service.requests)
	}
}

func TestRecoveryJobIgnoresUnsettledAndUnconfiguredProviders(t *testing.T) {
	service := &settlementStub{}
	job, err := NewRecoveryJob(recoveryRepoStub{candidates: []RecoveryCandidate{
		{PaymentID: uuid.New(), Provider: "stripe"}, {PaymentID: uuid.New(), Provider: "paystack"},
	}}, service, map[string]ProviderVerifier{"stripe": verifierStub{}}, 10, time.Second)
	if err != nil {
		t.Fatal(err)
	}
	if err := job.RunOnce(context.Background()); err != nil {
		t.Fatal(err)
	}
	if len(service.requests) != 0 {
		t.Fatalf("unexpected settlements: %+v", service.requests)
	}
}

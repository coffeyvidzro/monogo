package routing

import (
	"errors"
	"testing"
	"time"

	"github.com/google/uuid"
)

func TestRankCarriersBalancesCostQualityAndFailover(t *testing.T) {
	now := time.Date(2026, time.September, 25, 12, 0, 0, 0, time.UTC)
	policy := testOrchestrationPolicy()
	cheap := candidate(now, 10_000, 65, 150*time.Millisecond)
	quality := candidate(now, 10_500, 99, 20*time.Millisecond)
	backup := candidate(now, 12_000, 95, 50*time.Millisecond)

	ranked, err := RankCarriers(now, policy, []CarrierCandidate{backup, cheap, quality})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 3 {
		t.Fatalf("got %d routes, want three failover routes", len(ranked))
	}
	if ranked[0].Candidate.EndpointID != quality.EndpointID {
		t.Fatalf("poor quality route won on price: %+v", ranked)
	}
	if ranked[1].Candidate.EndpointID != backup.EndpointID || ranked[2].Candidate.EndpointID != cheap.EndpointID {
		t.Fatalf("unexpected failover order: %+v", ranked)
	}
}

func TestRankCarriersExcludesStaleUnhealthyAndBelowFloor(t *testing.T) {
	now := time.Now().UTC()
	policy := testOrchestrationPolicy()
	stale := candidate(now.Add(-2*time.Minute), 1, 99, 10*time.Millisecond)
	unhealthy := candidate(now, 1, 99, 10*time.Millisecond)
	unhealthy.Healthy = false
	poorASR := candidate(now, 1, 39, 10*time.Millisecond)
	highLoss := candidate(now, 1, 99, 10*time.Millisecond)
	highLoss.PacketLossPercent = 6

	_, err := RankCarriers(now, policy, []CarrierCandidate{stale, unhealthy, poorASR, highLoss})
	if !errors.Is(err, ErrNoEligibleCarrier) {
		t.Fatalf("got %v, want ErrNoEligibleCarrier", err)
	}
}

func TestRankCarriersIsDeterministicAndDeduplicatesEndpoints(t *testing.T) {
	now := time.Now().UTC()
	policy := testOrchestrationPolicy()
	a := candidate(now, 10_000, 90, 50*time.Millisecond)
	b := candidate(now, 10_000, 90, 50*time.Millisecond)
	duplicate := a
	duplicate.RateMicros = 1

	ranked, err := RankCarriers(now, policy, []CarrierCandidate{b, a, duplicate})
	if err != nil {
		t.Fatal(err)
	}
	if len(ranked) != 2 {
		t.Fatalf("got %d routes, want duplicate endpoint removed", len(ranked))
	}
	if ranked[0].Candidate.EndpointID.String() > ranked[1].Candidate.EndpointID.String() {
		t.Fatal("tie-break order is not deterministic")
	}
}

func TestRankCarriersPenalizesALOCBelowTargetButAboveFloor(t *testing.T) {
	now := time.Now().UTC()
	policy := testOrchestrationPolicy()
	shortCalls := candidate(now, 9_000, 95, 20*time.Millisecond)
	shortCalls.ALOCSeconds = 35
	longCalls := candidate(now, 10_000, 95, 20*time.Millisecond)

	ranked, err := RankCarriers(now, policy, []CarrierCandidate{shortCalls, longCalls})
	if err != nil {
		t.Fatal(err)
	}
	if ranked[0].Candidate.EndpointID != longCalls.EndpointID {
		t.Fatalf("low-ALOC route escaped its quality penalty: %+v", ranked)
	}
}

func testOrchestrationPolicy() OrchestrationPolicy {
	return OrchestrationPolicy{
		MaxSnapshotAge: time.Minute, MinASR: 40, MinALOCSeconds: 30,
		TargetALOCSeconds: 120,
		MaxLatency:        200 * time.Millisecond, MaxPacketLossPercent: 5,
		ASRPenaltyMicros: 10_000, ALOCPenaltyMicros: 5_000,
		LatencyPenaltyMicros: 2_000, LossPenaltyMicros: 10_000,
	}
}

func candidate(at time.Time, rate int64, asr float64, latency time.Duration) CarrierCandidate {
	return CarrierCandidate{
		CarrierConnectionID: uuid.New(), TrunkID: uuid.New(), EndpointID: uuid.New(),
		RateMicros: rate, ASR: asr, ALOCSeconds: 120, Latency: latency,
		PacketLossPercent: 0.1, Healthy: true, SnapshotAt: at,
	}
}

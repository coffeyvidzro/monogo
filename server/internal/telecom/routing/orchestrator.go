package routing

import (
	"errors"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/google/uuid"
)

// ErrNoEligibleCarrier means every route was stale, unhealthy, or below the
// configured quality floor. Callers should fail closed rather than route on
// incomplete pricing or telemetry.
var ErrNoEligibleCarrier = errors.New("no eligible carrier route")

// CarrierCandidate is the normalized snapshot consumed by the routing hot
// path. RateMicros is the wholesale price per minute in millionths of the
// billing currency; integer money avoids floating-point billing decisions.
type CarrierCandidate struct {
	CarrierConnectionID uuid.UUID
	TrunkID             uuid.UUID
	EndpointID          uuid.UUID
	RateMicros          int64
	ASR                 float64
	ALOCSeconds         float64
	Latency             time.Duration
	PacketLossPercent   float64
	Healthy             bool
	SnapshotAt          time.Time
}

// OrchestrationPolicy defines hard quality gates and the bounded penalty used
// to combine least-cost and quality routing. A penalty is expressed in rate
// micros, making the final score explainable and deterministic.
type OrchestrationPolicy struct {
	MaxSnapshotAge       time.Duration
	MinASR               float64
	MinALOCSeconds       float64
	TargetALOCSeconds    float64
	MaxLatency           time.Duration
	MaxPacketLossPercent float64
	ASRPenaltyMicros     int64
	ALOCPenaltyMicros    int64
	LatencyPenaltyMicros int64
	LossPenaltyMicros    int64
}

type RankedCarrier struct {
	Candidate CarrierCandidate
	Score     int64
}

func DefaultOrchestrationPolicy() OrchestrationPolicy {
	return OrchestrationPolicy{
		MaxSnapshotAge:       time.Minute,
		MinASR:               40,
		MinALOCSeconds:       30,
		TargetALOCSeconds:    120,
		MaxLatency:           200 * time.Millisecond,
		MaxPacketLossPercent: 5,
		ASRPenaltyMicros:     10_000,
		ALOCPenaltyMicros:    5_000,
		LatencyPenaltyMicros: 2_000,
		LossPenaltyMicros:    10_000,
	}
}

// RankCarriers returns the primary, secondary, and tertiary routes in failover
// order. Invalid telemetry is rejected, hard quality floors are applied first,
// and remaining routes are ranked by price plus quality penalties.
func RankCarriers(now time.Time, policy OrchestrationPolicy, candidates []CarrierCandidate) ([]RankedCarrier, error) {
	if err := validateOrchestrationPolicy(policy); err != nil {
		return nil, err
	}

	ranked := make([]RankedCarrier, 0, len(candidates))
	seen := make(map[uuid.UUID]struct{}, len(candidates))
	for _, candidate := range candidates {
		if err := validateCandidate(now, policy, candidate); err != nil {
			continue
		}
		if _, duplicate := seen[candidate.EndpointID]; duplicate {
			continue
		}
		score, err := carrierScore(policy, candidate)
		if err != nil {
			continue
		}
		seen[candidate.EndpointID] = struct{}{}
		ranked = append(ranked, RankedCarrier{
			Candidate: candidate,
			Score:     score,
		})
	}
	if len(ranked) == 0 {
		return nil, ErrNoEligibleCarrier
	}

	sort.Slice(ranked, func(i, j int) bool {
		if ranked[i].Score != ranked[j].Score {
			return ranked[i].Score < ranked[j].Score
		}
		if ranked[i].Candidate.RateMicros != ranked[j].Candidate.RateMicros {
			return ranked[i].Candidate.RateMicros < ranked[j].Candidate.RateMicros
		}
		return ranked[i].Candidate.EndpointID.String() < ranked[j].Candidate.EndpointID.String()
	})
	return ranked, nil
}

func validateOrchestrationPolicy(policy OrchestrationPolicy) error {
	if policy.MaxSnapshotAge <= 0 || policy.MaxLatency <= 0 {
		return fmt.Errorf("routing snapshot age and latency limits must be positive")
	}
	if !percentage(policy.MinASR) || !percentage(policy.MaxPacketLossPercent) ||
		policy.MinALOCSeconds < 0 || policy.TargetALOCSeconds < policy.MinALOCSeconds {
		return fmt.Errorf("routing quality limits are invalid")
	}
	if policy.ASRPenaltyMicros < 0 || policy.ALOCPenaltyMicros < 0 ||
		policy.LatencyPenaltyMicros < 0 || policy.LossPenaltyMicros < 0 {
		return fmt.Errorf("routing penalties cannot be negative")
	}
	return nil
}

func validateCandidate(now time.Time, policy OrchestrationPolicy, candidate CarrierCandidate) error {
	if candidate.CarrierConnectionID == uuid.Nil || candidate.TrunkID == uuid.Nil || candidate.EndpointID == uuid.Nil ||
		candidate.RateMicros < 0 || candidate.SnapshotAt.IsZero() || candidate.SnapshotAt.After(now) {
		return fmt.Errorf("invalid carrier candidate")
	}
	if !candidate.Healthy || now.Sub(candidate.SnapshotAt) > policy.MaxSnapshotAge {
		return fmt.Errorf("carrier health snapshot is ineligible")
	}
	if !percentage(candidate.ASR) || !percentage(candidate.PacketLossPercent) ||
		math.IsNaN(candidate.ALOCSeconds) || candidate.ALOCSeconds < policy.MinALOCSeconds ||
		candidate.ASR < policy.MinASR || candidate.Latency < 0 || candidate.Latency > policy.MaxLatency ||
		candidate.PacketLossPercent > policy.MaxPacketLossPercent {
		return fmt.Errorf("carrier quality is ineligible")
	}
	return nil
}

func percentage(value float64) bool { return !math.IsNaN(value) && value >= 0 && value <= 100 }

func carrierScore(policy OrchestrationPolicy, candidate CarrierCandidate) (int64, error) {
	asr := (100 - candidate.ASR) / 100
	aloc := 0.0
	if policy.TargetALOCSeconds > 0 {
		aloc = math.Max(0, policy.TargetALOCSeconds-candidate.ALOCSeconds) / policy.TargetALOCSeconds
	}
	latency := float64(candidate.Latency) / float64(policy.MaxLatency)
	loss := candidate.PacketLossPercent / 100
	score := float64(candidate.RateMicros) + asr*float64(policy.ASRPenaltyMicros) +
		aloc*float64(policy.ALOCPenaltyMicros) +
		latency*float64(policy.LatencyPenaltyMicros) +
		loss*float64(policy.LossPenaltyMicros)
	if math.IsInf(score, 0) || math.IsNaN(score) || score > math.MaxInt64 {
		return 0, fmt.Errorf("carrier score exceeds supported range")
	}
	return int64(math.Round(score)), nil
}

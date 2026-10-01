package session

import (
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type FailureReason string

const (
	FailureAttachTimeout       FailureReason = "attach_timeout"
	FailureInputBackpressure   FailureReason = "input_backpressure"
	FailureOutputBackpressure  FailureReason = "output_backpressure"
	FailureFrameTooLarge       FailureReason = "frame_too_large"
	FailureProviderTimeout     FailureReason = "provider_timeout"
	FailurePlaybackTimeout     FailureReason = "playback_timeout"
	FailureProvider            FailureReason = "provider_failure"
	FailureControlBackpressure FailureReason = "control_backpressure"
	FailureTransport           FailureReason = "transport_failure"
)

type MetricsSnapshot struct {
	ActiveSessions       uint64
	StartedSessions      uint64
	CompletedSessions    uint64
	AttachLatencyCount   uint64
	AttachLatencyTotalNS uint64
	TurnLatencyCount     uint64
	TurnLatencyTotalNS   uint64
	Failures             map[FailureReason]uint64
}

type Metrics struct {
	active        atomic.Uint64
	started       atomic.Uint64
	completed     atomic.Uint64
	attachCount   atomic.Uint64
	attachTotalNS atomic.Uint64
	turnCount     atomic.Uint64
	turnTotalNS   atomic.Uint64

	mu       sync.Mutex
	failures map[FailureReason]uint64
}

func newMetrics() *Metrics {
	return &Metrics{failures: make(map[FailureReason]uint64)}
}

func (m *Metrics) sessionStarted() {
	m.started.Add(1)
	m.active.Add(1)
}

func (m *Metrics) sessionRemoved() {
	for {
		current := m.active.Load()
		if current == 0 || m.active.CompareAndSwap(current, current-1) {
			return
		}
	}
}

func (m *Metrics) sessionCompleted() {
	m.completed.Add(1)
}

func (m *Metrics) observeAttachLatency(value time.Duration) {
	if value < 0 {
		return
	}
	m.attachCount.Add(1)
	m.attachTotalNS.Add(uint64(value))
}

func (m *Metrics) observeTurnLatency(value time.Duration) {
	if value < 0 {
		return
	}
	m.turnCount.Add(1)
	m.turnTotalNS.Add(uint64(value))
}

func (m *Metrics) failure(reason FailureReason) {
	m.mu.Lock()
	m.failures[reason]++
	m.mu.Unlock()
}

func (m *Metrics) snapshot() MetricsSnapshot {
	m.mu.Lock()
	failures := make(map[FailureReason]uint64, len(m.failures))
	for reason, count := range m.failures {
		failures[reason] = count
	}
	m.mu.Unlock()
	return MetricsSnapshot{
		ActiveSessions:       m.active.Load(),
		StartedSessions:      m.started.Load(),
		CompletedSessions:    m.completed.Load(),
		AttachLatencyCount:   m.attachCount.Load(),
		AttachLatencyTotalNS: m.attachTotalNS.Load(),
		TurnLatencyCount:     m.turnCount.Load(),
		TurnLatencyTotalNS:   m.turnTotalNS.Load(),
		Failures:             failures,
	}
}

func (s MetricsSnapshot) FailureReasons() []FailureReason {
	reasons := make([]FailureReason, 0, len(s.Failures))
	for reason := range s.Failures {
		reasons = append(reasons, reason)
	}
	sort.Slice(reasons, func(i, j int) bool { return reasons[i] < reasons[j] })
	return reasons
}

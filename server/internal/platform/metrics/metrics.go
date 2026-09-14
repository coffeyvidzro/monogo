package metrics

import (
	"context"
	"fmt"
	"strings"
	"sync/atomic"

	"github.com/google/uuid"
)

const maxTelecomSeries int64 = 10_000

type sharedStore interface {
	IncrementMetric(context.Context, string, int64) error
	SetMetricGauge(context.Context, string, float64, int64) error
	TelecomMetrics(context.Context) (map[string]string, map[string]string, error)
}

type Registry struct {
	requests atomic.Uint64
	errors   atomic.Uint64
	store    sharedStore
}

func New(store ...sharedStore) *Registry {
	r := &Registry{}
	if len(store) > 0 {
		r.store = store[0]
	}
	return r
}

func (r *Registry) IncRequests()     { r.requests.Add(1) }
func (r *Registry) IncErrors()       { r.errors.Add(1) }
func (r *Registry) Requests() uint64 { return r.requests.Load() }
func (r *Registry) Errors() uint64   { return r.errors.Load() }

func (r *Registry) Call(ctx context.Context, state string, carrier, trunk, endpoint uuid.UUID) {
	r.increment(ctx, series("calls_"+state+"_total", carrier, trunk, endpoint, ""))
}

func (r *Registry) LimitRejection(ctx context.Context, reason string, carrier uuid.UUID) {
	r.increment(ctx, series("limit_rejections_total", carrier, uuid.Nil, uuid.Nil, reason))
}

func (r *Registry) EndpointSelection(ctx context.Context, carrier, trunk, endpoint uuid.UUID, failover bool) {
	result := "primary"
	if failover {
		result = "failover"
	}
	r.increment(ctx, series("endpoint_selections_total", carrier, trunk, endpoint, result))
}

func (r *Registry) Probe(ctx context.Context, trunk, endpoint uuid.UUID, healthy bool, latencySeconds float64) {
	status := "failure"
	value := float64(0)
	if healthy {
		status, value = "success", 1
	}
	r.increment(ctx, series("endpoint_probes_total", uuid.Nil, trunk, endpoint, status))
	r.gauge(ctx, series("endpoint_probe_healthy", uuid.Nil, trunk, endpoint, ""), value)
	r.gauge(ctx, series("endpoint_probe_latency_seconds", uuid.Nil, trunk, endpoint, ""), latencySeconds)
}

func (r *Registry) increment(ctx context.Context, field string) {
	if r != nil && r.store != nil {
		_ = r.store.IncrementMetric(ctx, field, maxTelecomSeries)
	}
}
func (r *Registry) gauge(ctx context.Context, field string, value float64) {
	if r != nil && r.store != nil {
		_ = r.store.SetMetricGauge(ctx, field, value, maxTelecomSeries)
	}
}
func (r *Registry) Snapshot(ctx context.Context) (map[string]string, map[string]string, error) {
	if r == nil || r.store == nil {
		return map[string]string{}, map[string]string{}, nil
	}
	return r.store.TelecomMetrics(ctx)
}

func series(name string, carrier, trunk, endpoint uuid.UUID, result string) string {
	return strings.Join([]string{name, id(carrier), id(trunk), id(endpoint), result}, "|")
}
func id(value uuid.UUID) string {
	if value == uuid.Nil {
		return ""
	}
	return value.String()
}

func ParseSeries(field string) (string, [4]string, error) {
	parts := strings.Split(field, "|")
	if len(parts) != 5 || parts[0] == "" {
		return "", [4]string{}, fmt.Errorf("invalid telecom metric series")
	}
	return parts[0], [4]string{parts[1], parts[2], parts[3], parts[4]}, nil
}

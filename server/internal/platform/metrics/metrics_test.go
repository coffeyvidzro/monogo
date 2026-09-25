package metrics

import (
	"context"
	"testing"

	"github.com/google/uuid"
)

type metricStoreStub struct{ field string }

func (s *metricStoreStub) IncrementMetric(_ context.Context, field string, _ int64) error {
	s.field = field
	return nil
}
func (*metricStoreStub) SetMetricGauge(context.Context, string, float64, int64) error { return nil }
func (*metricStoreStub) TelecomMetrics(context.Context) (map[string]string, map[string]string, error) {
	return nil, nil, nil
}

func TestRouteAttemptMetricIncludesRouteOutcomeAndAttempt(t *testing.T) {
	store := &metricStoreStub{}
	carrier, trunk, endpoint := uuid.New(), uuid.New(), uuid.New()
	New(store).RouteAttempt(
		context.Background(), carrier, trunk, endpoint,
		"retryable_failure", "sip", 2,
	)
	name, labels, err := ParseSeries(store.field)
	if err != nil {
		t.Fatal(err)
	}
	if name != "route_attempts_total" || labels[0] != carrier.String() ||
		labels[1] != trunk.String() || labels[2] != endpoint.String() ||
		labels[3] != "retryable_failure:sip:2" {
		t.Fatalf("unexpected metric series: %q", store.field)
	}
}

package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type HealthObservation struct {
	EndpointID            uuid.UUID
	ASRBasisPoints        int32
	ALOCMilliseconds      int64
	LatencyMilliseconds   int32
	PacketLossBasisPoints int32
	SampleCount           int64
	ObservedAt            time.Time
}

type SnapshotIngestor struct {
	queries *sqlc.Queries
	now     func() time.Time
}

func NewSnapshotIngestor(queries *sqlc.Queries) *SnapshotIngestor {
	return &SnapshotIngestor{
		queries: queries,
		now:     time.Now,
	}
}

func validateHealthObservation(observation HealthObservation, now time.Time) error {
	if observation.EndpointID == uuid.Nil || observation.SampleCount < 1 {
		return fmt.Errorf("health observation requires an endpoint and measured sample")
	}
	if observation.ObservedAt.IsZero() ||
		observation.ObservedAt.After(now) ||
		now.Sub(observation.ObservedAt) > time.Minute {
		return fmt.Errorf("health observation is stale or has an invalid timestamp")
	}
	if observation.ASRBasisPoints < 0 ||
		observation.ASRBasisPoints > 10_000 ||
		observation.ALOCMilliseconds < 0 ||
		observation.LatencyMilliseconds < 0 ||
		observation.PacketLossBasisPoints < 0 ||
		observation.PacketLossBasisPoints > 10_000 {
		return fmt.Errorf("health observation contains invalid quality values")
	}
	return nil
}

// IngestHealth accepts measured snapshots only. The endpoint's independently
// managed health_status and its eligibility policy remain separate controls.
func (i *SnapshotIngestor) IngestHealth(
	ctx context.Context,
	observation HealthObservation,
) (sqlc.CarrierRouteMetric, error) {
	if i == nil || i.queries == nil {
		return sqlc.CarrierRouteMetric{}, fmt.Errorf("routing snapshot storage is unavailable")
	}
	if err := validateHealthObservation(observation, i.now().UTC()); err != nil {
		return sqlc.CarrierRouteMetric{}, err
	}
	return i.queries.UpsertCarrierRouteMetrics(
		ctx,
		sqlc.UpsertCarrierRouteMetricsParams{
			TrunkEndpointID:       observation.EndpointID,
			AsrBasisPoints:        observation.ASRBasisPoints,
			AlocMilliseconds:      observation.ALOCMilliseconds,
			LatencyMilliseconds:   observation.LatencyMilliseconds,
			PacketLossBasisPoints: observation.PacketLossBasisPoints,
			SampleCount:           observation.SampleCount,
			ObservedAt:            pgconv.TimeToTimestamptz(observation.ObservedAt),
		},
	)
}

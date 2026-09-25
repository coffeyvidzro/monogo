package routing

import (
	"context"
	"fmt"
	"regexp"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgtype"
)

var destinationPrefix = regexp.MustCompile(`^[1-9][0-9]{0,14}$`)

// RateObservation must come from a verified, effective-dated carrier rate sheet.
// A number-inventory response or an unrated CDR is not a rate observation.
type RateObservation struct {
	CarrierConnectionID uuid.UUID
	DestinationPrefix   string
	Currency            string
	RateMicros          int64
	EffectiveAt         time.Time
	ExpiresAt           *time.Time
}

// HealthObservation represents measured, non-synthetic SIP/RTP route telemetry.
// A collector must supply endpoint identity and an actual observation time.
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

func validateRateObservation(observation RateObservation) error {
	if observation.CarrierConnectionID == uuid.Nil {
		return fmt.Errorf("rate observation requires a carrier connection")
	}
	if !destinationPrefix.MatchString(observation.DestinationPrefix) {
		return fmt.Errorf("rate observation requires a valid destination prefix")
	}
	if observation.Currency != "USD" || observation.RateMicros < 0 {
		return fmt.Errorf("rate observation requires nonnegative USD micros")
	}
	if observation.EffectiveAt.IsZero() {
		return fmt.Errorf("rate observation requires an effective timestamp")
	}
	if observation.ExpiresAt != nil &&
		!observation.ExpiresAt.After(observation.EffectiveAt) {
		return fmt.Errorf("rate observation expiry must follow its effective timestamp")
	}
	return nil
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

// IngestRate accepts an authenticated upstream rate observation after the
// provider-specific collector has verified its schema, account, and currency.
func (i *SnapshotIngestor) IngestRate(
	ctx context.Context,
	observation RateObservation,
) (sqlc.CarrierRate, error) {
	if i == nil || i.queries == nil {
		return sqlc.CarrierRate{}, fmt.Errorf("routing snapshot storage is unavailable")
	}
	if err := validateRateObservation(observation); err != nil {
		return sqlc.CarrierRate{}, err
	}
	expiresAt := pgtype.Timestamptz{}
	if observation.ExpiresAt != nil {
		expiresAt = pgconv.TimeToTimestamptz(*observation.ExpiresAt)
	}
	return i.queries.CreateCarrierRate(ctx, sqlc.CreateCarrierRateParams{
		CarrierConnectionID: observation.CarrierConnectionID,
		DestinationPrefix:   observation.DestinationPrefix,
		RateMicros:          observation.RateMicros,
		BillingCurrency:     observation.Currency,
		EffectiveAt:         pgconv.TimeToTimestamptz(observation.EffectiveAt),
		ExpiresAt:           expiresAt,
	})
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

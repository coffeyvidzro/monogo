package routing

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

func NewRepository(queries *sqlc.Queries, db *pgxpool.Pool) *Repository {
	return &Repository{queries: queries, db: db}
}

func (r *Repository) GetInboundContext(
	ctx context.Context,
	req InboundRequest,
) (Limits, error) {
	binding, err := r.queries.GetVoiceBindingByID(ctx, sqlc.GetVoiceBindingByIDParams{
		ID:             req.VoiceBindingID,
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		return Limits{}, err
	}
	if binding.VoiceApplicationID != req.ApplicationID ||
		binding.PhoneNumberID == nil ||
		*binding.PhoneNumberID != req.PhoneNumberID {
		return Limits{}, pgx.ErrNoRows
	}

	carrierConnectionID := req.CarrierConnectionID
	row, err := r.queries.GetInboundCallContext(ctx, sqlc.GetInboundCallContextParams{
		PhoneNumberID:       req.PhoneNumberID,
		OrganizationID:      req.OrganizationID,
		CalledNumber:        req.CalledNumber,
		CarrierConnectionID: &carrierConnectionID,
		ApplicationID:       req.ApplicationID,
	})
	if err != nil {
		return Limits{}, err
	}
	return Limits{
		MaxCPS:             row.MaxCps,
		MaxConcurrentCalls: row.MaxConcurrentCalls,
		MaxDailyMinutes:    row.MaxDailyMinutes,
	}, nil
}

func (r *Repository) ResolveBYOCOutbound(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
) (OutboundRoute, error) {
	trunk, err := r.queries.GetTrunkByID(ctx, sqlc.GetTrunkByIDParams{
		ID:             trunkID,
		OrganizationID: &organizationID,
	})
	if err != nil {
		return OutboundRoute{}, err
	}
	if trunk.ProvisioningMode != "byoc" || trunk.CarrierConnectionID == nil {
		return OutboundRoute{}, pgx.ErrNoRows
	}

	connection, err := r.queries.GetCarrierConnectionByID(ctx, sqlc.GetCarrierConnectionByIDParams{
		ID:             *trunk.CarrierConnectionID,
		OrganizationID: &organizationID,
	})
	if err != nil {
		return OutboundRoute{}, err
	}

	endpoints, err := r.queries.ListActiveOutboundTrunkEndpoints(
		ctx,
		sqlc.ListActiveOutboundTrunkEndpointsParams{
			TrunkID:        trunkID,
			OrganizationID: &organizationID,
		},
	)
	if err != nil {
		return OutboundRoute{}, err
	}

	for _, endpoint := range endpoints {
		if endpoint.HealthStatus == "unhealthy" {
			continue
		}
		if endpoint.Port < 1 || endpoint.Port > 65535 {
			return OutboundRoute{}, fmt.Errorf("invalid trunk endpoint port: %d", endpoint.Port)
		}
		return OutboundRoute{
			CarrierConnectionID: *trunk.CarrierConnectionID,
			TrunkID:             trunk.ID,
			TrunkEndpointID:     endpoint.ID,
			ProvisioningMode:    trunk.ProvisioningMode,
			Host:                endpoint.Host,
			Port:                uint16(endpoint.Port),
			Transport:           endpoint.Transport,
			Limits: Limits{
				MaxCPS:             connection.MaxCps,
				MaxConcurrentCalls: connection.MaxConcurrentCalls,
				MaxDailyMinutes:    connection.MaxDailyMinutes,
			},
		}, nil
	}

	return OutboundRoute{}, pgx.ErrNoRows
}

func (r *Repository) ListManagedOutboundCandidates(
	ctx context.Context,
	destinationDigits string,
	resolvedAt time.Time,
) ([]managedRouteCandidate, error) {
	rows, err := r.queries.ListManagedRouteCandidates(ctx, sqlc.ListManagedRouteCandidatesParams{
		DestinationDigits: destinationDigits,
		ResolvedAt:        pgconv.TimeToTimestamptz(resolvedAt),
	})
	if err != nil {
		return nil, err
	}

	candidates := make([]managedRouteCandidate, 0, len(rows))
	for _, row := range rows {
		if row.Port < 1 || row.Port > 65535 {
			return nil, fmt.Errorf("invalid managed endpoint port: %d", row.Port)
		}
		observedAt := pgconv.TimestamptzToTime(row.ObservedAt)
		candidates = append(candidates, managedRouteCandidate{
			Candidate: CarrierCandidate{
				CarrierConnectionID: row.CarrierConnectionID,
				TrunkID:             row.TrunkID,
				EndpointID:          row.TrunkEndpointID,
				RateMicros:          row.RateMicros,
				ASR:                 float64(row.AsrBasisPoints) / 100,
				ALOCSeconds:         float64(row.AlocMilliseconds) / 1000,
				Latency:             time.Duration(row.LatencyMilliseconds) * time.Millisecond,
				PacketLossPercent:   float64(row.PacketLossBasisPoints) / 100,
				Healthy:             row.HealthStatus == "healthy",
				SnapshotAt:          observedAt,
			},
			Route: OutboundRoute{
				CarrierConnectionID: row.CarrierConnectionID,
				TrunkID:             row.TrunkID,
				TrunkEndpointID:     row.TrunkEndpointID,
				ProvisioningMode:    "managed",
				Host:                row.Host,
				Port:                uint16(row.Port),
				Transport:           row.Transport,
				RateMicros:          row.RateMicros,
				Limits: Limits{
					MaxCPS:             row.MaxCps,
					MaxConcurrentCalls: row.MaxConcurrentCalls,
					MaxDailyMinutes:    row.MaxDailyMinutes,
				},
			},
			ASRBasisPoints:        row.AsrBasisPoints,
			ALOCMilliseconds:      row.AlocMilliseconds,
			LatencyMilliseconds:   row.LatencyMilliseconds,
			PacketLossBasisPoints: row.PacketLossBasisPoints,
			MetricsObservedAt:     observedAt,
		})
	}
	return candidates, nil
}

func (r *Repository) RecordManagedDecision(
	ctx context.Context,
	organizationID uuid.UUID,
	destination string,
	ranked []RankedCarrier,
	candidates map[uuid.UUID]managedRouteCandidate,
) (uuid.UUID, error) {
	if r.db == nil {
		return uuid.Nil, fmt.Errorf("routing decision database is required")
	}
	tx, err := r.db.BeginTx(ctx, pgx.TxOptions{})
	if err != nil {
		return uuid.Nil, err
	}
	defer func() { _ = tx.Rollback(ctx) }()
	queries := r.queries.WithTx(tx)
	selected := ranked[0].Candidate
	decision, err := queries.CreateRoutingDecision(ctx, sqlc.CreateRoutingDecisionParams{
		OrganizationID: organizationID, Destination: destination,
		SelectedCarrierConnectionID: selected.CarrierConnectionID,
		SelectedTrunkID:             selected.TrunkID, SelectedTrunkEndpointID: selected.EndpointID,
		CandidateCount: int32(len(ranked)), // #nosec G115 -- route plans are bounded by configured endpoints.
	})
	if err != nil {
		return uuid.Nil, err
	}
	for index, rankedCandidate := range ranked {
		candidate := candidates[rankedCandidate.Candidate.EndpointID]
		err = queries.CreateRoutingDecisionCandidate(ctx, sqlc.CreateRoutingDecisionCandidateParams{
			RoutingDecisionID: decision.ID, Rank: int32(index + 1), // #nosec G115 -- route plans are bounded by configured endpoints.
			CarrierConnectionID: candidate.Candidate.CarrierConnectionID,
			TrunkID:             candidate.Candidate.TrunkID, TrunkEndpointID: candidate.Candidate.EndpointID,
			RateMicros: candidate.Candidate.RateMicros, AsrBasisPoints: candidate.ASRBasisPoints,
			AlocMilliseconds: candidate.ALOCMilliseconds, LatencyMilliseconds: candidate.LatencyMilliseconds,
			PacketLossBasisPoints: candidate.PacketLossBasisPoints,
			MetricsObservedAt:     pgconv.TimeToTimestamptz(candidate.MetricsObservedAt),
			ScoreMicros:           rankedCandidate.Score,
		})
		if err != nil {
			return uuid.Nil, err
		}
	}
	if err := tx.Commit(ctx); err != nil {
		return uuid.Nil, err
	}
	return decision.ID, nil
}

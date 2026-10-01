package routing

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	queries *sqlc.Queries
	db      *pgxpool.Pool
}

type inboundContext struct {
	Limits Limits
}

func NewRepository(
	queries *sqlc.Queries,
	db *pgxpool.Pool,
) *Repository {
	return &Repository{queries: queries, db: db}
}

func (r *Repository) GetInboundContext(
	ctx context.Context,
	req InboundRequest,
) (inboundContext, error) {
	binding, err := r.queries.GetVoiceAgentBindingByID(
		ctx,
		sqlc.GetVoiceAgentBindingByIDParams{
			ID:             req.VoiceAgentBindingID,
			OrganizationID: req.OrganizationID,
		},
	)
	if err != nil {
		return inboundContext{}, err
	}
	if binding.VoiceAgentID != req.VoiceAgentID ||
		binding.PhoneNumberID != req.PhoneNumberID {
		return inboundContext{}, pgx.ErrNoRows
	}

	row, err := r.queries.GetInboundCallContext(
		ctx,
		sqlc.GetInboundCallContextParams{
			PhoneNumberID:       req.PhoneNumberID,
			OrganizationID:      req.OrganizationID,
			CalledNumber:        req.CalledNumber,
			TrunkID:             &req.TrunkID,
			VoiceAgentBindingID: req.VoiceAgentBindingID,
			VoiceAgentID:        req.VoiceAgentID,
		},
	)
	if err != nil {
		return inboundContext{}, err
	}

	return inboundContext{
		Limits: Limits{
			MaxCPS:             row.MaxCps,
			MaxConcurrentCalls: row.MaxConcurrentCalls,
		},
	}, nil
}

func (r *Repository) ResolveBYOCOutbound(
	ctx context.Context,
	organizationID, trunkID uuid.UUID,
) ([]OutboundRoute, error) {
	trunk, err := r.queries.GetTrunkByID(
		ctx,
		sqlc.GetTrunkByIDParams{
			ID:             trunkID,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return nil, err
	}
	if trunk.Status != "active" ||
		(trunk.Direction != "outbound" &&
			trunk.Direction != "bidirectional") {
		return nil, pgx.ErrNoRows
	}

	endpoints, err := r.queries.ListActiveOutboundTrunkEndpoints(
		ctx,
		sqlc.ListActiveOutboundTrunkEndpointsParams{
			TrunkID:        trunkID,
			OrganizationID: organizationID,
		},
	)
	if err != nil {
		return nil, err
	}

	routes := make([]OutboundRoute, 0, len(endpoints))
	for _, endpoint := range endpoints {
		if endpoint.HealthStatus == "unhealthy" {
			continue
		}
		if endpoint.Port < 1 || endpoint.Port > 65535 {
			return nil, fmt.Errorf(
				"invalid trunk endpoint port: %d",
				endpoint.Port,
			)
		}
		routes = append(routes, OutboundRoute{
			TrunkID:         trunk.ID,
			TrunkEndpointID: endpoint.ID,
			Host:            endpoint.Host,
			Port:            uint16(endpoint.Port),
			Transport:       endpoint.Transport,
			Limits: Limits{
				MaxCPS:             trunk.MaxCps,
				MaxConcurrentCalls: trunk.MaxConcurrentCalls,
			},
		})
	}

	if len(routes) == 0 {
		return nil, pgx.ErrNoRows
	}
	return routes, nil
}

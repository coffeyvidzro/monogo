package routing

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetInboundContext(
	ctx context.Context,
	req InboundRequest,
) (Limits, error) {
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
) (OutboundDecision, error) {
	trunk, err := r.queries.GetTrunkByID(ctx, sqlc.GetTrunkByIDParams{
		ID:             trunkID,
		OrganizationID: &organizationID,
	})
	if err != nil {
		return OutboundDecision{}, err
	}
	if trunk.ProvisioningMode != "byoc" || trunk.CarrierConnectionID == nil {
		return OutboundDecision{}, pgx.ErrNoRows
	}

	endpoints, err := r.queries.ListActiveOutboundTrunkEndpoints(
		ctx,
		sqlc.ListActiveOutboundTrunkEndpointsParams{
			TrunkID:        trunkID,
			OrganizationID: &organizationID,
		},
	)
	if err != nil {
		return OutboundDecision{}, err
	}

	for _, endpoint := range endpoints {
		if endpoint.HealthStatus == "unhealthy" {
			continue
		}
		if endpoint.Port < 1 || endpoint.Port > 65535 {
			return OutboundDecision{}, fmt.Errorf("invalid trunk endpoint port: %d", endpoint.Port)
		}
		return OutboundDecision{
			CarrierConnectionID: *trunk.CarrierConnectionID,
			TrunkID:             trunk.ID,
			TrunkEndpointID:     endpoint.ID,
			ProvisioningMode:    trunk.ProvisioningMode,
			Host:                endpoint.Host,
			Port:                uint16(endpoint.Port),
			Transport:           endpoint.Transport,
		}, nil
	}

	return OutboundDecision{}, pgx.ErrNoRows
}

func (r *Repository) ResolveManagedOutbound(ctx context.Context) (OutboundDecision, error) {
	routes, err := r.queries.ResolveManagedOutboundRoute(ctx)
	if err != nil {
		return OutboundDecision{}, err
	}

	for _, route := range routes {
		if route.HealthStatus == "unhealthy" {
			continue
		}
		if route.Port < 1 || route.Port > 65535 {
			return OutboundDecision{}, fmt.Errorf("invalid managed endpoint port: %d", route.Port)
		}
		return OutboundDecision{
			CarrierConnectionID: route.CarrierConnectionID,
			TrunkID:             route.TrunkID,
			TrunkEndpointID:     route.EndpointID,
			ProvisioningMode:    "managed",
			Host:                route.Host,
			Port:                uint16(route.Port),
			Transport:           route.Transport,
		}, nil
	}

	return OutboundDecision{}, pgx.ErrNoRows
}

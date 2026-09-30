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

func NewRepository(queries *sqlc.Queries, db *pgxpool.Pool) *Repository {
	return &Repository{queries: queries, db: db}
}

func (r *Repository) GetInboundContext(
	ctx context.Context,
	req InboundRequest,
) (inboundContext, error) {
	binding, err := r.queries.GetVoiceBindingByID(ctx, sqlc.GetVoiceBindingByIDParams{
		ID:             req.VoiceBindingID,
		OrganizationID: req.OrganizationID,
	})
	if err != nil {
		return inboundContext{}, err
	}
	if binding.VoiceApplicationID != req.ApplicationID ||
		binding.PhoneNumberID == nil ||
		*binding.PhoneNumberID != req.PhoneNumberID {
		return inboundContext{}, pgx.ErrNoRows
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
		return inboundContext{}, err
	}
	return inboundContext{
		Limits: Limits{
			MaxCPS:             row.MaxCps,
			MaxConcurrentCalls: row.MaxConcurrentCalls,
			MaxDailyMinutes:    row.MaxDailyMinutes,
		},
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
	if trunk.CarrierConnectionID == nil {
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


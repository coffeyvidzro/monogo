package routing

import (
	"context"
	"net/netip"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type managedRouteCandidate struct {
	CarrierConnectionID uuid.UUID
	MaxCPS              int32
	MaxConcurrentCalls  int32
	MaxDailyMinutes     *int64
	TrunkID             uuid.UUID
	EndpointID          uuid.UUID
	Host                string
	Port                int32
	Transport           string
	Priority            int32
	Weight              int32
	HealthStatus        string
}

type Repository struct {
	queries *sqlc.Queries
}

func NewRepository(queries *sqlc.Queries) *Repository {
	return &Repository{queries: queries}
}

func (r *Repository) GetTrunk(
	ctx context.Context,
	organizationID uuid.UUID,
	trunkID uuid.UUID,
) (sqlc.Trunk, error) {
	return r.queries.GetTrunkByID(ctx, sqlc.GetTrunkByIDParams{
		ID:             trunkID,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) GetCarrierConnection(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.CarrierConnection, error) {
	row, err := r.queries.GetCarrierConnectionByID(ctx, sqlc.GetCarrierConnectionByIDParams{
		ID:             id,
		OrganizationID: &organizationID,
	})
	if err != nil {
		return sqlc.CarrierConnection{}, err
	}
	return sqlc.CarrierConnection{
		ID:                 row.ID,
		OrganizationID:     row.OrganizationID,
		ProviderID:         row.ProviderID,
		Scope:              row.Scope,
		Status:             row.Status,
		MaxCps:             row.MaxCps,
		MaxConcurrentCalls: row.MaxConcurrentCalls,
		MaxDailyMinutes:    row.MaxDailyMinutes,
	}, nil
}

func (r *Repository) ListOutboundEndpoints(
	ctx context.Context,
	organizationID uuid.UUID,
	trunkID uuid.UUID,
) ([]sqlc.TrunkEndpoint, error) {
	return r.queries.ListActiveOutboundTrunkEndpoints(ctx, sqlc.ListActiveOutboundTrunkEndpointsParams{
		TrunkID:        trunkID,
		OrganizationID: &organizationID,
	})
}

func (r *Repository) ListManagedOutboundRoutes(ctx context.Context) ([]managedRouteCandidate, error) {
	rows, err := r.queries.ResolveManagedOutboundRoute(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]managedRouteCandidate, 0, len(rows))
	for _, row := range rows {
		result = append(result, managedRouteCandidate{
			CarrierConnectionID: row.CarrierConnectionID,
			MaxCPS:              row.MaxCps,
			MaxConcurrentCalls:  row.MaxConcurrentCalls,
			MaxDailyMinutes:     row.MaxDailyMinutes,
			TrunkID:             row.TrunkID,
			EndpointID:          row.EndpointID,
			Host:                row.Host,
			Port:                row.Port,
			Transport:           row.Transport,
			Priority:            row.Priority,
			Weight:              row.Weight,
			HealthStatus:        row.HealthStatus,
		})
	}
	return result, nil
}

func (r *Repository) ResolveInboundCarrier(
	ctx context.Context,
	sourceIP netip.Addr,
) (sqlc.CarrierConnection, error) {
	return r.queries.ResolveCarrierConnectionBySourceIPAnyScope(ctx, sourceIP)
}

func (r *Repository) GetPhoneNumber(
	ctx context.Context,
	organizationID uuid.UUID,
	number string,
) (sqlc.PhoneNumber, error) {
	return r.queries.GetPhoneNumberByNumber(ctx, sqlc.GetPhoneNumberByNumberParams{
		Number:         number,
		OrganizationID: organizationID,
	})
}

func (r *Repository) ResolveInboundPhoneNumber(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	number string,
) (sqlc.PhoneNumber, error) {
	return r.queries.ResolveInboundPhoneNumber(ctx, sqlc.ResolveInboundPhoneNumberParams{
		CarrierConnectionID: &carrierConnectionID,
		Number:              number,
	})
}

func (r *Repository) GetVoiceBinding(
	ctx context.Context,
	number string,
) (sqlc.GetVoiceBindingByNumberRow, error) {
	return r.queries.GetVoiceBindingByNumber(ctx, number)
}

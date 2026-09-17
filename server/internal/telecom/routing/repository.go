package routing

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
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

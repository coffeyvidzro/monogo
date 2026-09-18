package calls

import (
	"context"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
)

type Repository struct {
	queries *sqlc.Queries
}

type LifecycleSnapshot = sqlc.GetCallLifecycleSnapshotRow

func NewRepository(queries *sqlc.Queries) *Repository {
	if queries == nil {
		panic("calls: queries are required")
	}
	return &Repository{queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (sqlc.Call, error) {
	return r.queries.CreateCall(ctx, sqlc.CreateCallParams{
		OrganizationID: organizationID,
		ApplicationID:  req.ApplicationID,
		Direction:      string(req.Direction),
		State:          nil,
		FromUri:        req.FromURI,
		ToUri:          req.ToURI,
		SipCallID:      req.SIPCallID,
	})
}

func (r *Repository) CreateInbound(
	ctx context.Context,
	req InboundAdmissionRequest,
) (sqlc.Call, error) {
	state := string(StateRinging)
	applicationID := req.ApplicationID
	sipCallID := req.SIPCallID
	return r.queries.CreateCall(ctx, sqlc.CreateCallParams{
		OrganizationID: req.OrganizationID,
		ApplicationID:  &applicationID,
		Direction:      string(DirectionInbound),
		State:          &state,
		FromUri:        req.FromURI,
		ToUri:          req.ToURI,
		SipCallID:      &sipCallID,
	})
}

func (r *Repository) GetBySIPCallIDGlobal(
	ctx context.Context,
	sipCallID string,
) (sqlc.Call, error) {
	return r.queries.GetCallBySIPCallIDGlobal(ctx, &sipCallID)
}

func (r *Repository) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Call, error) {
	return r.queries.GetCall(ctx, sqlc.GetCallParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) GetLifecycleSnapshot(
	ctx context.Context,
	id uuid.UUID,
) (LifecycleSnapshot, error) {
	return r.queries.GetCallLifecycleSnapshot(ctx, id)
}

func (r *Repository) CarrierDailyUsageSeconds(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
) (int64, error) {
	return r.queries.GetCarrierDailyUsageSeconds(ctx, carrierConnectionID)
}

func (r *Repository) List(
	ctx context.Context,
	organizationID uuid.UUID,
	req ListRequest,
) ([]sqlc.Call, error) {
	return r.queries.ListCalls(ctx, sqlc.ListCallsParams{
		OrganizationID: organizationID,
		State:          req.State,
		PageOffset:     req.Offset,
		PageLimit:      req.Limit,
	})
}

func (r *Repository) SetRouteAttribution(
	ctx context.Context,
	organizationID, id uuid.UUID,
	route RouteAttribution,
) (sqlc.Call, error) {
	return r.queries.SetCallRouteAttribution(ctx, sqlc.SetCallRouteAttributionParams{
		CarrierConnectionID: route.CarrierConnectionID,
		TrunkID:             route.TrunkID,
		TrunkEndpointID:     route.TrunkEndpointID,
		OrganizationID:      organizationID,
		ID:                  id,
	})
}

func (r *Repository) MarkRinging(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.queries.MarkCallRinging(ctx, sqlc.MarkCallRingingParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkAnswered(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.queries.MarkCallAnswered(ctx, sqlc.MarkCallAnsweredParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkActive(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.queries.MarkCallActive(ctx, sqlc.MarkCallActiveParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkHeld(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.queries.MarkCallHeld(ctx, sqlc.MarkCallHeldParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkResumed(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.queries.MarkCallResumed(ctx, sqlc.MarkCallResumedParams{
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkCompleted(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.queries.MarkCallCompleted(ctx, sqlc.MarkCallCompletedParams{
		HangupReason:   reason,
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.queries.MarkCallFailed(ctx, sqlc.MarkCallFailedParams{
		HangupReason:   reason,
		OrganizationID: organizationID,
		ID:             id,
	})
}

func (r *Repository) MarkCancelled(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.queries.MarkCallCancelled(ctx, sqlc.MarkCallCancelledParams{
		HangupReason:   reason,
		OrganizationID: organizationID,
		ID:             id,
	})
}

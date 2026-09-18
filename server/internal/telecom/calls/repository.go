package calls

import (
	"context"
	"fmt"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

type LifecycleSnapshot struct {
	OrganizationID      uuid.UUID
	CarrierConnectionID *uuid.UUID
	Direction           string
	State               string
	MediaState          string
}

func NewRepository(db *pgxpool.Pool, queries *sqlc.Queries) *Repository {
	if db == nil {
		panic("calls: database is required")
	}
	if queries == nil {
		panic("calls: queries are required")
	}
	return &Repository{db: db, queries: queries}
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
	const query = `
SELECT
    organization_id,
    carrier_connection_id,
    direction,
    state,
    media_state
FROM calls
WHERE id = $1
LIMIT 1
`

	var snapshot LifecycleSnapshot
	if err := r.db.QueryRow(ctx, query, id).Scan(
		&snapshot.OrganizationID,
		&snapshot.CarrierConnectionID,
		&snapshot.Direction,
		&snapshot.State,
		&snapshot.MediaState,
	); err != nil {
		return LifecycleSnapshot{}, err
	}
	return snapshot, nil
}

// CarrierDailyUsageSeconds returns answered carrier usage overlapping the
// current UTC calendar day. Active calls contribute elapsed answered time up
// to NOW(), so new admissions see already-consumed in-flight usage.
func (r *Repository) CarrierDailyUsageSeconds(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
) (int64, error) {
	if carrierConnectionID == uuid.Nil {
		return 0, fmt.Errorf("carrier connection id is required")
	}

	const query = `
WITH bounds AS (
    SELECT
        date_trunc('day', NOW() AT TIME ZONE 'UTC') AT TIME ZONE 'UTC' AS day_start,
        (date_trunc('day', NOW() AT TIME ZONE 'UTC') + INTERVAL '1 day') AT TIME ZONE 'UTC' AS day_end
)
SELECT COALESCE(
    SUM(
        GREATEST(
            EXTRACT(
                EPOCH FROM (
                    LEAST(COALESCE(c.ended_at, NOW()), b.day_end)
                    - GREATEST(c.answered_at, b.day_start)
                )
            ),
            0
        )
    ),
    0
)::BIGINT
FROM calls AS c
CROSS JOIN bounds AS b
WHERE c.carrier_connection_id = $1
  AND c.answered_at IS NOT NULL
  AND c.answered_at < b.day_end
  AND COALESCE(c.ended_at, NOW()) > b.day_start
`

	var seconds int64
	if err := r.db.QueryRow(ctx, query, carrierConnectionID).Scan(&seconds); err != nil {
		return 0, fmt.Errorf("query carrier daily usage: %w", err)
	}
	return seconds, nil
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

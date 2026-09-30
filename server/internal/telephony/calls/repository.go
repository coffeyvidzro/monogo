package calls

import (
	"context"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/platform/outbox"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5/pgxpool"
)

type Repository struct {
	db      *pgxpool.Pool
	queries *sqlc.Queries
}

type LifecycleSnapshot = sqlc.GetCallLifecycleSnapshotRow

func NewRepository(queries *sqlc.Queries, db *pgxpool.Pool) *Repository {
	if queries == nil {
		panic("calls: queries are required")
	}
	if db == nil {
		panic("calls: database is required")
	}
	return &Repository{db: db, queries: queries}
}

func (r *Repository) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (sqlc.Call, error) {
	return r.mutate(ctx, "call.initiated", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.CreateCall(ctx, sqlc.CreateCallParams{
			OrganizationID: organizationID,
			ApplicationID:  req.ApplicationID,
			Direction:      string(req.Direction),
			State:          nil,
			FromUri:        req.FromURI,
			ToUri:          req.ToURI,
			SipCallID:      req.SIPCallID,
		})
	})
}

func (r *Repository) CreateInbound(
	ctx context.Context,
	req InboundAdmissionRequest,
) (sqlc.Call, error) {
	state := string(StateRinging)
	applicationID := req.ApplicationID
	sipCallID := req.SIPCallID
	return r.mutate(ctx, "call.ringing", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.CreateCall(ctx, sqlc.CreateCallParams{
			OrganizationID: req.OrganizationID,
			ApplicationID:  &applicationID,
			Direction:      string(DirectionInbound),
			State:          &state,
			FromUri:        req.FromURI,
			ToUri:          req.ToURI,
			SipCallID:      &sipCallID,
		})
	})
}

// BindOutboundSIPCallID records the first real SIP dialog identity observed on
// the originating FreeSWITCH channel. It never substitutes the logical call ID
// or FreeSWITCH channel UUID for the SIP Call-ID.
func (r *Repository) BindOutboundSIPCallID(ctx context.Context, organizationID, callID uuid.UUID, sipCallID string) error {
	_, err := r.queries.SetOutboundCallSIPCallID(ctx, sqlc.SetOutboundCallSIPCallIDParams{
		SipCallID:      &sipCallID,
		OrganizationID: organizationID,
		ID:             callID,
	})
	return err
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
	return r.queries.GetCarrierDailyUsageSeconds(ctx, &carrierConnectionID)
}

func (r *Repository) ListActiveForAdmissionReconciliation(
	ctx context.Context,
) ([]ActiveAdmissionCall, error) {
	rows, err := r.queries.ListActiveCallsForAdmissionReconciliation(ctx)
	if err != nil {
		return nil, err
	}

	items := make([]ActiveAdmissionCall, 0, len(rows))
	for _, row := range rows {
		if row.CarrierConnectionID == nil {
			continue
		}
		items = append(items, ActiveAdmissionCall{
			ID:                  row.ID,
			CarrierConnectionID: *row.CarrierConnectionID,
		})
	}
	return items, nil
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
		RoutingDecisionID:   route.RoutingDecisionID,
		OrganizationID:      organizationID,
		ID:                  id,
	})
}

func (r *Repository) RecordRoutingAttempt(
	ctx context.Context,
	decisionID, callID uuid.UUID,
	attempt routeAttemptOutcome,
) error {
	var failureClass *string
	if attempt.FailureClass != "" {
		failureClass = &attempt.FailureClass
	}
	var sipStatus *int32
	if attempt.SIPStatus != 0 {
		value := int32(attempt.SIPStatus) // #nosec G115 -- validated SIP status is between 100 and 699.
		sipStatus = &value
	}
	_, err := r.queries.CreateRoutingAttempt(ctx, sqlc.CreateRoutingAttemptParams{
		RoutingDecisionID:    decisionID,
		CallID:               callID,
		Attempt:              int32(attempt.Attempt), // #nosec G115 -- failover is bounded to three attempts.
		CarrierConnectionID:  attempt.Route.CarrierConnectionID,
		TrunkID:              attempt.Route.TrunkID,
		TrunkEndpointID:      attempt.Route.TrunkEndpointID,
		Outcome:              attempt.Outcome,
		FailureClass:         failureClass,
		SipStatus:            sipStatus,
		DurationMilliseconds: attempt.Duration.Milliseconds(),
	})
	return err
}

func (r *Repository) SetRoutingDecisionSelectedRoute(
	ctx context.Context,
	organizationID, decisionID uuid.UUID,
	route routing.OutboundRoute,
) error {
	rows, err := r.queries.SetRoutingDecisionSelectedRoute(ctx, sqlc.SetRoutingDecisionSelectedRouteParams{
		CarrierConnectionID: route.CarrierConnectionID,
		TrunkID:             route.TrunkID,
		TrunkEndpointID:     route.TrunkEndpointID,
		ID:                  decisionID,
		OrganizationID:      organizationID,
	})
	if err != nil {
		return err
	}
	if rows != 1 {
		return fmt.Errorf("routing decision selection update affected %d rows", rows)
	}
	return nil
}

func (r *Repository) MarkRinging(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.mutate(ctx, "call.ringing", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallRinging(ctx, sqlc.MarkCallRingingParams{
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkAnswered(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.mutate(ctx, "call.answered", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallAnswered(ctx, sqlc.MarkCallAnsweredParams{
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkActive(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.mutate(ctx, "call.active", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallActive(ctx, sqlc.MarkCallActiveParams{
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkHeld(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.mutate(ctx, "call.held", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallHeld(ctx, sqlc.MarkCallHeldParams{
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkResumed(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return r.mutate(ctx, "call.resumed", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallResumed(ctx, sqlc.MarkCallResumedParams{
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkCompleted(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.mutate(ctx, "call.completed", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallCompleted(ctx, sqlc.MarkCallCompletedParams{
			HangupReason:   reason,
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkFailed(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.mutate(ctx, "call.failed", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallFailed(ctx, sqlc.MarkCallFailedParams{
			HangupReason:   reason,
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

func (r *Repository) MarkCancelled(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	return r.mutate(ctx, "call.cancelled", func(queries *sqlc.Queries) (sqlc.Call, error) {
		return queries.MarkCallCancelled(ctx, sqlc.MarkCallCancelledParams{
			HangupReason:   reason,
			OrganizationID: organizationID,
			ID:             id,
		})
	})
}

type callMutation func(*sqlc.Queries) (sqlc.Call, error)

// mutate commits the call state and its corresponding outbox event atomically.
// Failed or replayed lifecycle transitions do not emit duplicate events.
func (r *Repository) mutate(
	ctx context.Context,
	eventType string,
	mutation callMutation,
) (sqlc.Call, error) {
	tx, err := r.db.Begin(ctx)
	if err != nil {
		return sqlc.Call{}, fmt.Errorf("begin call transaction: %w", err)
	}
	defer func() { _ = tx.Rollback(ctx) }()

	queries := r.queries.WithTx(tx)
	call, err := mutation(queries)
	if err != nil {
		return sqlc.Call{}, err
	}

	occurredAt := time.Now().UTC()
	if _, err := outbox.NewRepository(queries).Insert(ctx, outbox.Event{
		Subject:       eventType,
		AggregateType: "call",
		AggregateID:   call.ID,
		Payload: map[string]any{
			"event_type":      eventType,
			"organization_id": call.OrganizationID,
			"call_id":         call.ID,
			"resource":        callResponse(call),
			"occurred_at":     occurredAt,
		},
		Headers: map[string]string{
			"event_type":      eventType,
			"organization_id": call.OrganizationID.String(),
			"schema_version":  "1",
		},
	}); err != nil {
		return sqlc.Call{}, fmt.Errorf("insert call outbox event: %w", err)
	}

	if err := tx.Commit(ctx); err != nil {
		return sqlc.Call{}, fmt.Errorf("commit call transaction: %w", err)
	}
	return call, nil
}

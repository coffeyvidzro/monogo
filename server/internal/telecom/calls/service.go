package calls

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo *Repository
}

func NewService(repo *Repository) *Service {
	if repo == nil {
		panic("calls: repository is required")
	}
	return &Service{repo: repo}
}

func (s *Service) Create(
	ctx context.Context,
	organizationID uuid.UUID,
	req CreateRequest,
) (sqlc.Call, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.Call{}, err
	}

	req, err := normalizeCreateRequest(req)
	if err != nil {
		return sqlc.Call{}, err
	}

	call, err := s.repo.Create(ctx, organizationID, req)
	if err != nil {
		return sqlc.Call{}, apperror.NewInternal("create call", err)
	}
	return call, nil
}

func (s *Service) Get(
	ctx context.Context,
	organizationID, id uuid.UUID,
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}

	call, err := s.repo.Get(ctx, organizationID, id)
	return call, translateReadError(err)
}

func (s *Service) List(
	ctx context.Context,
	organizationID uuid.UUID,
	req ListRequest,
) ([]sqlc.Call, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	if err := validateListRequest(req); err != nil {
		return nil, err
	}

	items, err := s.repo.List(ctx, organizationID, req)
	if err != nil {
		return nil, apperror.NewInternal("list calls", err)
	}
	return items, nil
}

func (s *Service) SetRouteAttribution(
	ctx context.Context,
	organizationID, id uuid.UUID,
	route RouteAttribution,
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	if err := validateRouteAttribution(route); err != nil {
		return sqlc.Call{}, err
	}

	call, err := s.repo.SetRouteAttribution(ctx, organizationID, id, route)
	return call, translateMutationError(err)
}

func (s *Service) ObserveLifecycle(ctx context.Context, event LifecycleEvent) error {
	if event.CallID == uuid.Nil {
		return apperror.NewBadRequest("call lifecycle event requires call id")
	}

	snapshot, err := s.repo.GetLifecycleSnapshot(ctx, event.CallID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("resolve lifecycle call", err)
	}

	if lifecycleAlreadyApplied(snapshot, event.Type) {
		return nil
	}

	switch event.Type {
	case LifecycleInitiated:
		return nil
	case LifecycleRinging:
		_, err = s.MarkRinging(ctx, snapshot.OrganizationID, event.CallID)
	case LifecycleAnswered:
		_, err = s.MarkAnswered(ctx, snapshot.OrganizationID, event.CallID)
	case LifecycleActive:
		_, err = s.MarkActive(ctx, snapshot.OrganizationID, event.CallID)
	case LifecycleHeld:
		_, err = s.MarkHeld(ctx, snapshot.OrganizationID, event.CallID)
	case LifecycleResumed:
		_, err = s.MarkResumed(ctx, snapshot.OrganizationID, event.CallID)
	case LifecycleCompleted:
		_, err = s.MarkCompleted(ctx, snapshot.OrganizationID, event.CallID, event.HangupReason)
	case LifecycleFailed:
		_, err = s.MarkFailed(ctx, snapshot.OrganizationID, event.CallID, event.HangupReason)
	case LifecycleCancelled:
		_, err = s.MarkCancelled(ctx, snapshot.OrganizationID, event.CallID, event.HangupReason)
	default:
		return apperror.NewBadRequest("unsupported call lifecycle event")
	}
	return err
}

func lifecycleAlreadyApplied(snapshot LifecycleSnapshot, eventType LifecycleEventType) bool {
	switch eventType {
	case LifecycleInitiated:
		return true
	case LifecycleRinging:
		return snapshot.State != string(StateInitiating)
	case LifecycleAnswered:
		return snapshot.State == string(StateAnswered) ||
			snapshot.State == string(StateActive) ||
			isTerminalState(snapshot.State)
	case LifecycleActive:
		return snapshot.State == string(StateActive) || isTerminalState(snapshot.State)
	case LifecycleHeld:
		return snapshot.MediaState == string(MediaStateHeld) || isTerminalState(snapshot.State)
	case LifecycleResumed:
		return snapshot.MediaState == string(MediaStateActive) || isTerminalState(snapshot.State)
	case LifecycleCompleted, LifecycleFailed, LifecycleCancelled:
		return isTerminalState(snapshot.State)
	default:
		return false
	}
}

func isTerminalState(state string) bool {
	return state == string(StateCompleted) ||
		state == string(StateFailed) ||
		state == string(StateCancelled)
}

func (s *Service) MarkRinging(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return s.transition(ctx, organizationID, id, s.repo.MarkRinging)
}

func (s *Service) MarkAnswered(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return s.transition(ctx, organizationID, id, s.repo.MarkAnswered)
}

func (s *Service) MarkActive(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return s.transition(ctx, organizationID, id, s.repo.MarkActive)
}

func (s *Service) MarkHeld(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return s.transition(ctx, organizationID, id, s.repo.MarkHeld)
}

func (s *Service) MarkResumed(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	return s.transition(ctx, organizationID, id, s.repo.MarkResumed)
}

func (s *Service) MarkCompleted(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkCompleted(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}

func (s *Service) MarkFailed(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkFailed(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}

func (s *Service) MarkCancelled(
	ctx context.Context,
	organizationID, id uuid.UUID,
	reason *string,
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkCancelled(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}

func (s *Service) transition(
	ctx context.Context,
	organizationID, id uuid.UUID,
	fn func(context.Context, uuid.UUID, uuid.UUID) (sqlc.Call, error),
) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	call, err := fn(ctx, organizationID, id)
	return call, translateMutationError(err)
}

func translateReadError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("call not found")
	}
	return apperror.NewInternal("get call", err)
}

func translateMutationError(err error) error {
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewConflict("call state transition or route attribution is not allowed")
	}
	return apperror.NewInternal("update call", err)
}

package calls

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) ObserveLifecycle(ctx context.Context, event LifecycleEvent) error {
	if event.CallID == uuid.Nil {
		return apperror.NewBadRequest("call lifecycle event requires call id")
	}
	if event.ChannelID != "" && !isTerminalLifecycle(event.Type) {
		if err := s.channels.Bind(ctx, event.CallID, event.ChannelID); err != nil {
			return apperror.NewInternal("bind call channel", err)
		}
	}

	snapshot, err := s.repo.GetLifecycleSnapshot(ctx, event.CallID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return apperror.NewInternal("resolve lifecycle call", err)
	}
	if !isTerminalLifecycle(event.Type) && snapshot.CarrierConnectionID != nil {
		if err := s.admission.Refresh(ctx, *snapshot.CarrierConnectionID, event.CallID); err != nil {
			return apperror.NewServiceUnavailable("refresh carrier call lease", err)
		}
	}
	if lifecycleAlreadyApplied(snapshot, event.Type) {
		if isTerminalLifecycle(event.Type) {
			_ = s.channels.Delete(ctx, event.CallID)
			if snapshot.CarrierConnectionID != nil {
				_ = s.admission.Release(ctx, *snapshot.CarrierConnectionID, event.CallID.String())
			}
		}
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
	if err == nil && isTerminalLifecycle(event.Type) {
		_ = s.channels.Delete(ctx, event.CallID)
		if snapshot.CarrierConnectionID != nil {
			_ = s.admission.Release(ctx, *snapshot.CarrierConnectionID, event.CallID.String())
		}
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
		return snapshot.State == string(StateAnswered) || snapshot.State == string(StateActive) || isTerminalState(snapshot.State)
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

func isTerminalLifecycle(eventType LifecycleEventType) bool {
	return eventType == LifecycleCompleted || eventType == LifecycleFailed || eventType == LifecycleCancelled
}

func isTerminalState(state string) bool {
	return state == string(StateCompleted) || state == string(StateFailed) || state == string(StateCancelled)
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

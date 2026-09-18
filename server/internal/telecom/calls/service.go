package calls

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo       *Repository
	router     *routing.Service
	controller Controller
	channels   ChannelStore
}

func NewService(repo *Repository, router *routing.Service, controller Controller, channels ChannelStore) *Service {
	if repo == nil {
		panic("calls: repository is required")
	}
	if router == nil {
		panic("calls: routing service is required")
	}
	if controller == nil {
		panic("calls: controller is required")
	}
	if channels == nil {
		panic("calls: channel store is required")
	}
	return &Service{repo: repo, router: router, controller: controller, channels: channels}
}

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateRequest) (sqlc.Call, error) {
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

	decision, err := s.router.ResolveOutbound(ctx, routing.OutboundRequest{
		OrganizationID: organizationID,
		TrunkID:        req.TrunkID,
		Destination:    req.ToURI,
	})
	if err != nil {
		reason := "route_resolution_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, err
	}

	carrierID, trunkID, endpointID := decision.CarrierConnectionID, decision.TrunkID, decision.TrunkEndpointID
	call, err = s.repo.SetRouteAttribution(ctx, organizationID, call.ID, RouteAttribution{
		CarrierConnectionID: &carrierID,
		TrunkID:             &trunkID,
		TrunkEndpointID:     &endpointID,
	})
	if err != nil {
		reason := "route_attribution_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, apperror.NewInternal("set call route attribution", err)
	}

	result, err := s.controller.Originate(ctx, OriginateRequest{
		CallID:              call.ID,
		Destination:         req.ToURI,
		CallerID:            req.FromURI,
		CarrierConnectionID: decision.CarrierConnectionID,
		Host:                decision.Host,
		Port:                decision.Port,
		Transport:           decision.Transport,
		Privacy:             req.Privacy,
		DTMFMode:            req.DTMFMode,
		MediaEncryption:     req.MediaEncryption,
	})
	if err != nil {
		reason := "originate_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, apperror.NewInternal("originate call", err)
	}
	if err := s.channels.Bind(ctx, call.ID, result.ChannelID); err != nil {
		_ = s.controller.Hangup(ctx, result.ChannelID)
		reason := "channel_binding_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, apperror.NewInternal("bind call channel", err)
	}

	return s.repo.Get(ctx, organizationID, call.ID)
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil {
		return sqlc.Call{}, err
	}
	call, err := s.repo.Get(ctx, organizationID, id)
	return call, translateReadError(err)
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, req ListRequest) ([]sqlc.Call, error) {
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

func (s *Service) SetRouteAttribution(ctx context.Context, organizationID, id uuid.UUID, route RouteAttribution) (sqlc.Call, error) {
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
	if lifecycleAlreadyApplied(snapshot, event.Type) {
		if isTerminalLifecycle(event.Type) {
			_ = s.channels.Delete(ctx, event.CallID)
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
	}
	return err
}

func (s *Service) Answer(ctx context.Context, org, id uuid.UUID) error {
	return s.control(ctx, org, id, []State{StateInitiating, StateRinging}, func(channelID string) error {
		return s.controller.Answer(ctx, channelID)
	})
}

func (s *Service) Hangup(ctx context.Context, org, id uuid.UUID) error {
	call, channelID, err := s.controlContext(ctx, org, id)
	if err != nil {
		return err
	}
	if isTerminalState(call.State) {
		return nil
	}
	if err := s.controller.Hangup(ctx, channelID); err != nil {
		return apperror.NewInternal("hangup call", err)
	}
	return nil
}

func (s *Service) Transfer(ctx context.Context, org, id uuid.UUID, req TransferActionRequest) error {
	req, err := normalizeTransfer(req)
	if err != nil {
		return err
	}
	return s.control(ctx, org, id, []State{StateAnswered, StateActive}, func(channelID string) error {
		return s.controller.Transfer(ctx, channelID, TransferRequest{Destination: req.Destination})
	})
}

func (s *Service) Hold(ctx context.Context, org, id uuid.UUID) error {
	call, channelID, err := s.controlContext(ctx, org, id)
	if err != nil {
		return err
	}
	if call.State != string(StateAnswered) && call.State != string(StateActive) {
		return apperror.NewConflict("call cannot be held in its current state")
	}
	if call.MediaState == string(MediaStateHeld) {
		return nil
	}
	if err := s.controller.Hold(ctx, channelID); err != nil {
		return apperror.NewInternal("hold call", err)
	}
	return nil
}

func (s *Service) Resume(ctx context.Context, org, id uuid.UUID) error {
	call, channelID, err := s.controlContext(ctx, org, id)
	if err != nil {
		return err
	}
	if call.State != string(StateAnswered) && call.State != string(StateActive) {
		return apperror.NewConflict("call cannot be resumed in its current state")
	}
	if call.MediaState == string(MediaStateActive) {
		return nil
	}
	if err := s.controller.Resume(ctx, channelID); err != nil {
		return apperror.NewInternal("resume call", err)
	}
	return nil
}

func (s *Service) Play(ctx context.Context, org, id uuid.UUID, req PlayActionRequest) error {
	req, err := normalizePlay(req)
	if err != nil {
		return err
	}
	return s.control(ctx, org, id, []State{StateAnswered, StateActive}, func(channelID string) error {
		return s.controller.PlayAudio(ctx, channelID, req.Path)
	})
}

func (s *Service) Stop(ctx context.Context, org, id uuid.UUID) error {
	return s.control(ctx, org, id, []State{StateAnswered, StateActive}, func(channelID string) error {
		return s.controller.StopPlayback(ctx, channelID)
	})
}

func (s *Service) Record(ctx context.Context, org, id uuid.UUID, req RecordActionRequest) error {
	req, err := normalizeRecord(req)
	if err != nil {
		return err
	}
	return s.control(ctx, org, id, []State{StateAnswered, StateActive}, func(channelID string) error {
		return s.controller.Record(ctx, channelID, RecordRequest{Path: req.Path, Action: req.Action})
	})
}

func (s *Service) DTMF(ctx context.Context, org, id uuid.UUID, req DTMFActionRequest) error {
	req, err := normalizeDTMF(req)
	if err != nil {
		return err
	}
	return s.control(ctx, org, id, []State{StateAnswered, StateActive}, func(channelID string) error {
		return s.controller.SendDTMF(ctx, channelID, req.Digits)
	})
}

func (s *Service) control(ctx context.Context, org, id uuid.UUID, allowed []State, fn func(string) error) error {
	call, channelID, err := s.controlContext(ctx, org, id)
	if err != nil {
		return err
	}
	ok := false
	for _, state := range allowed {
		if call.State == string(state) {
			ok = true
			break
		}
	}
	if !ok {
		return apperror.NewConflict("call control is not allowed in the current state")
	}
	if err := fn(channelID); err != nil {
		return apperror.NewInternal("control call", err)
	}
	return nil
}

func (s *Service) controlContext(ctx context.Context, org, id uuid.UUID) (sqlc.Call, string, error) {
	call, err := s.Get(ctx, org, id)
	if err != nil {
		return sqlc.Call{}, "", err
	}
	channelID, err := s.channels.Get(ctx, id)
	if errors.Is(err, ErrChannelUnavailable) {
		return sqlc.Call{}, "", apperror.NewConflict("call has no active media channel")
	}
	if err != nil {
		return sqlc.Call{}, "", apperror.NewInternal("resolve call channel", err)
	}
	return call, channelID, nil
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
func (s *Service) MarkCompleted(ctx context.Context, organizationID, id uuid.UUID, reason *string) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.Call{}, err }
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkCompleted(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}
func (s *Service) MarkFailed(ctx context.Context, organizationID, id uuid.UUID, reason *string) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.Call{}, err }
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkFailed(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}
func (s *Service) MarkCancelled(ctx context.Context, organizationID, id uuid.UUID, reason *string) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.Call{}, err }
	reason = normalizeOptionalReason(reason)
	call, err := s.repo.MarkCancelled(ctx, organizationID, id, reason)
	return call, translateMutationError(err)
}
func (s *Service) transition(ctx context.Context, organizationID, id uuid.UUID, fn func(context.Context, uuid.UUID, uuid.UUID) (sqlc.Call, error)) (sqlc.Call, error) {
	if err := validateIDs(organizationID, id); err != nil { return sqlc.Call{}, err }
	call, err := fn(ctx, organizationID, id)
	return call, translateMutationError(err)
}
func translateReadError(err error) error {
	if err == nil { return nil }
	if errors.Is(err, pgx.ErrNoRows) { return apperror.NewNotFound("call not found") }
	return apperror.NewInternal("get call", err)
}
func translateMutationError(err error) error {
	if err == nil { return nil }
	if errors.Is(err, pgx.ErrNoRows) { return apperror.NewConflict("call state transition or route attribution is not allowed") }
	return apperror.NewInternal("update call", err)
}


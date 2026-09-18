package calls

import (
	"context"
	"errors"

	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/internal/runtime/calling"
	"github.com/coffeyvidzro/monogo/internal/telecom/routing"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type Service struct {
	repo       *Repository
	router     *routing.Service
	controller *calling.Controller
	channels   *calling.ChannelStore
	admission  *calling.AdmissionLimiter
}

func NewService(
	repo *Repository,
	router *routing.Service,
	controller *calling.Controller,
	channels *calling.ChannelStore,
	admission *calling.AdmissionLimiter,
) *Service {
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
	if admission == nil {
		panic("calls: admission limiter is required")
	}
	return &Service{
		repo:       repo,
		router:     router,
		controller: controller,
		channels:   channels,
		admission:  admission,
	}
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

	if err := s.checkDailyMinutes(ctx, decision.CarrierConnectionID, decision.Limits.MaxDailyMinutes); err != nil {
		reason := admissionFailureReason(err)
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, admissionError(err)
	}

	if err := s.admission.Acquire(
		ctx,
		decision.CarrierConnectionID,
		call.ID.String(),
		decision.Limits,
	); err != nil {
		reason := admissionFailureReason(err)
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, admissionError(err)
	}

	result, err := s.controller.Originate(ctx, calling.OriginateRequest{
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
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, call.ID.String())
		reason := "originate_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, apperror.NewInternal("originate call", err)
	}
	if err := s.channels.Bind(ctx, call.ID, result.ChannelID); err != nil {
		_ = s.controller.Hangup(ctx, result.ChannelID)
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, call.ID.String())
		reason := "channel_binding_failed"
		_, _ = s.repo.MarkFailed(ctx, organizationID, call.ID, &reason)
		return sqlc.Call{}, apperror.NewInternal("bind call channel", err)
	}

	return s.repo.Get(ctx, organizationID, call.ID)
}

func (s *Service) AdmitInbound(
	ctx context.Context,
	req InboundAdmissionRequest,
) (sqlc.Call, error) {
	if err := validateInboundAdmission(req); err != nil {
		return sqlc.Call{}, err
	}

	existing, err := s.repo.GetBySIPCallIDGlobal(ctx, req.SIPCallID)
	if err == nil {
		if err := validateExistingInbound(existing, req); err != nil {
			return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, err)
		}
		existing, err = s.ensureInboundAttribution(ctx, existing, req)
		if err != nil {
			return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, err)
		}
		if existing.CarrierConnectionID != nil {
			if err := s.admission.Refresh(ctx, *existing.CarrierConnectionID, existing.ID); err != nil {
				return sqlc.Call{}, s.rejectInbound(
					ctx,
					req.ChannelID,
					apperror.NewServiceUnavailable("refresh inbound admission lease", err),
				)
			}
		}
		if err := s.bindInboundChannel(ctx, existing, req.ChannelID); err != nil {
			return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, err)
		}
		return existing, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Call{}, s.rejectInbound(
			ctx,
			req.ChannelID,
			apperror.NewInternal("lookup inbound SIP call", err),
		)
	}

	decision, err := s.router.ResolveInbound(ctx, routing.InboundRequest{
		OrganizationID:      req.OrganizationID,
		ApplicationID:       req.ApplicationID,
		PhoneNumberID:       req.PhoneNumberID,
		VoiceBindingID:      req.VoiceBindingID,
		CarrierConnectionID: req.CarrierConnectionID,
		CalledNumber:        req.ToURI,
	})
	if err != nil {
		return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, err)
	}

	if err := s.checkDailyMinutes(ctx, decision.CarrierConnectionID, decision.Limits.MaxDailyMinutes); err != nil {
		return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, admissionError(err))
	}

	if err := s.admission.Acquire(
		ctx,
		decision.CarrierConnectionID,
		req.ChannelID,
		decision.Limits,
	); err != nil {
		return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, admissionError(err))
	}

	call, err := s.repo.CreateInbound(ctx, req)
	if err != nil {
		// A duplicate CHANNEL_CREATE or a concurrent admission can race the
		// unique SIP Call-ID constraint. Re-read and reuse the durable call.
		existing, readErr := s.repo.GetBySIPCallIDGlobal(ctx, req.SIPCallID)
		if readErr == nil {
			if existing.ID.String() != req.ChannelID {
				_ = s.admission.Release(ctx, decision.CarrierConnectionID, req.ChannelID)
			}
			if validateErr := validateExistingInbound(existing, req); validateErr != nil {
				return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, validateErr)
			}
			existing, attributionErr := s.ensureInboundAttribution(ctx, existing, req)
			if attributionErr != nil {
				return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, attributionErr)
			}
			if existing.CarrierConnectionID != nil {
				if refreshErr := s.admission.Refresh(ctx, *existing.CarrierConnectionID, existing.ID); refreshErr != nil {
					return sqlc.Call{}, s.rejectInbound(
						ctx,
						req.ChannelID,
						apperror.NewServiceUnavailable("refresh inbound admission lease", refreshErr),
					)
				}
			}
			if bindErr := s.bindInboundChannel(ctx, existing, req.ChannelID); bindErr != nil {
				return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, bindErr)
			}
			return existing, nil
		}
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, req.ChannelID)
		return sqlc.Call{}, s.rejectInbound(
			ctx,
			req.ChannelID,
			apperror.NewInternal("create inbound call", err),
		)
	}

	carrierID := req.CarrierConnectionID
	call, err = s.repo.SetRouteAttribution(ctx, req.OrganizationID, call.ID, RouteAttribution{
		CarrierConnectionID: &carrierID,
	})
	if err != nil {
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, req.ChannelID)
		reason := "inbound_attribution_failed"
		_, _ = s.repo.MarkFailed(ctx, req.OrganizationID, call.ID, &reason)
		return sqlc.Call{}, s.rejectInbound(
			ctx,
			req.ChannelID,
			apperror.NewInternal("set inbound route attribution", err),
		)
	}

	if err := s.admission.Bind(
		ctx,
		decision.CarrierConnectionID,
		req.ChannelID,
		call.ID,
	); err != nil {
		reason := "inbound_admission_binding_failed"
		_, _ = s.repo.MarkFailed(ctx, req.OrganizationID, call.ID, &reason)
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, req.ChannelID)
		return sqlc.Call{}, s.rejectInbound(
			ctx,
			req.ChannelID,
			apperror.NewInternal("bind inbound admission lease", err),
		)
	}

	if err := s.bindInboundChannel(ctx, call, req.ChannelID); err != nil {
		_ = s.admission.Release(ctx, decision.CarrierConnectionID, call.ID.String())
		reason := "inbound_channel_binding_failed"
		_, _ = s.repo.MarkFailed(ctx, req.OrganizationID, call.ID, &reason)
		return sqlc.Call{}, s.rejectInbound(ctx, req.ChannelID, err)
	}

	return call, nil
}

func validateExistingInbound(call sqlc.Call, req InboundAdmissionRequest) error {
	if call.Direction != string(DirectionInbound) ||
		call.OrganizationID != req.OrganizationID ||
		call.ApplicationID == nil ||
		*call.ApplicationID != req.ApplicationID ||
		call.ToUri != req.ToURI {
		return apperror.NewConflict("SIP Call-ID is already associated with a different call")
	}
	if call.CarrierConnectionID != nil && *call.CarrierConnectionID != req.CarrierConnectionID {
		return apperror.NewConflict("SIP Call-ID carrier attribution does not match")
	}
	if isTerminalState(call.State) {
		return apperror.NewConflict("inbound SIP call is already terminal")
	}
	return nil
}

func (s *Service) ensureInboundAttribution(
	ctx context.Context,
	call sqlc.Call,
	req InboundAdmissionRequest,
) (sqlc.Call, error) {
	if call.CarrierConnectionID != nil {
		return call, nil
	}
	carrierID := req.CarrierConnectionID
	updated, err := s.repo.SetRouteAttribution(ctx, req.OrganizationID, call.ID, RouteAttribution{
		CarrierConnectionID: &carrierID,
	})
	if err != nil {
		return sqlc.Call{}, apperror.NewInternal("set inbound route attribution", err)
	}
	return updated, nil
}

func (s *Service) bindInboundChannel(
	ctx context.Context,
	call sqlc.Call,
	channelID string,
) error {
	if err := s.channels.Bind(ctx, call.ID, channelID); err != nil {
		return apperror.NewInternal("bind inbound call channel", err)
	}
	if err := s.controller.SetCallID(ctx, channelID, call.ID); err != nil {
		_ = s.channels.Delete(ctx, call.ID)
		return apperror.NewInternal("correlate inbound FreeSWITCH channel", err)
	}
	return nil
}

func (s *Service) rejectInbound(ctx context.Context, channelID string, cause error) error {
	if err := s.controller.Hangup(ctx, channelID); err != nil {
		return errors.Join(cause, err)
	}
	return cause
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
		return s.controller.Transfer(ctx, channelID, calling.TransferRequest{Destination: req.Destination})
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
		return s.controller.Record(ctx, channelID, calling.RecordRequest{Path: req.Path, Action: req.Action})
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
	if errors.Is(err, calling.ErrChannelUnavailable) {
		return sqlc.Call{}, "", apperror.NewConflict("call has no active media channel")
	}
	if err != nil {
		return sqlc.Call{}, "", apperror.NewInternal("resolve call channel", err)
	}
	return call, channelID, nil
}

func (s *Service) checkDailyMinutes(
	ctx context.Context,
	carrierConnectionID uuid.UUID,
	maxDailyMinutes *int64,
) error {
	if maxDailyMinutes == nil {
		return nil
	}
	usedSeconds, err := s.repo.CarrierDailyUsageSeconds(ctx, carrierConnectionID)
	if err != nil {
		return apperror.NewServiceUnavailable("read carrier daily usage", err)
	}
	if usedSeconds >= *maxDailyMinutes*60 {
		return ErrAdmissionDailyMinutes
	}
	return nil
}

func admissionError(err error) error {
	switch {
	case errors.Is(err, calling.ErrAdmissionCPS):
		return apperror.NewTooManyRequests("carrier CPS limit exceeded")
	case errors.Is(err, calling.ErrAdmissionConcurrent):
		return apperror.NewTooManyRequests("carrier concurrent call limit exceeded")
	case errors.Is(err, ErrAdmissionDailyMinutes):
		return apperror.NewTooManyRequests("carrier daily minute limit exceeded")
	default:
		return apperror.NewServiceUnavailable("carrier admission service unavailable", err)
	}
}

func admissionFailureReason(err error) string {
	switch {
	case errors.Is(err, calling.ErrAdmissionCPS):
		return "carrier_cps_limit"
	case errors.Is(err, calling.ErrAdmissionConcurrent):
		return "carrier_concurrent_limit"
	case errors.Is(err, ErrAdmissionDailyMinutes):
		return "carrier_daily_minutes_limit"
	default:
		return "carrier_admission_failed"
	}
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

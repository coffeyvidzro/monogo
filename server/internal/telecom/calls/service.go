package calls

import (
	"context"
	"errors"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/leamout/leamout/internal/database/sqlc"
	"github.com/leamout/leamout/internal/telecom/routing"
	"github.com/leamout/leamout/pkg/apperror"
)

// Controller is the media-server contract used by the call API.
type Controller interface {
	Originate(context.Context, OriginateRequest) (string, error)
	Answer(context.Context, string) error
	Hangup(context.Context, string) error
	Transfer(context.Context, string, TransferRequest) error
	Hold(context.Context, string) error
	Unhold(context.Context, string) error
	Play(context.Context, string, string) error
	Stop(context.Context, string) error
	Record(context.Context, string, RecordRequest) error
	DTMF(context.Context, string, string) error
}

type RouteResolver interface {
	ResolveOutbound(context.Context, routing.OutboundRequest) (routing.OutboundDecision, error)
}

type TelecomMetrics interface {
	Call(context.Context, string, uuid.UUID, uuid.UUID, uuid.UUID)
	LimitRejection(context.Context, string, uuid.UUID)
}

type Service struct {
	repo       *Repository
	controller Controller
	routes     RouteResolver
	admission  *AdmissionController
	metrics    TelecomMetrics
}

func NewService(repo *Repository, controller Controller, routes RouteResolver, admission ...*AdmissionController) *Service {
	if repo == nil {
		panic("calls: repository is required")
	}
	if controller == nil {
		panic("calls: controller is required")
	}
	if routes == nil {
		panic("calls: route resolver is required")
	}

	service := &Service{repo: repo, controller: controller, routes: routes}
	if len(admission) > 0 {
		service.admission = admission[0]
	}
	return service
}

func (s *Service) SetMetrics(metrics TelecomMetrics) { s.metrics = metrics }

func (s *Service) Create(ctx context.Context, organizationID uuid.UUID, req CreateCallRequest) (sqlc.Call, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.Call{}, err
	}
	if err := validateCreateRequest(&req); err != nil {
		return sqlc.Call{}, err
	}

	route, err := s.routes.ResolveOutbound(ctx, routing.OutboundRequest{
		OrganizationID: organizationID,
		ApplicationID:  req.ApplicationID,
		From:           req.From,
		To:             req.To,
		TrunkID:        req.TrunkID,
	})
	if err != nil {
		return sqlc.Call{}, routeError(err)
	}
	s.recordCall(ctx, "attempted", route.CarrierConnectionID, route.TrunkID, route.EndpointID)
	leaseID := ""
	if s.admission != nil {
		leaseID, err = s.admission.Admit(ctx, CallLimits{
			CarrierConnectionID: route.CarrierConnectionID,
			MaxCPS:              route.MaxCPS,
			MaxConcurrent:       route.MaxConcurrentCalls,
			MaxDailyMinutes:     route.MaxDailyMinutes,
		})
		if err != nil {
			s.recordLimitRejection(ctx, err, route.CarrierConnectionID)
			return sqlc.Call{}, admissionError(err)
		}
		defer func() {
			if leaseID != "" {
				_ = s.admission.Release(context.WithoutCancel(ctx), route.CarrierConnectionID, leaseID)
			}
		}()
	}

	sipCallID, err := s.controller.Originate(ctx, OriginateRequest{
		CarrierConnectionID: route.CarrierConnectionID,
		Host:                route.Host,
		Port:                route.Port,
		Transport:           route.Transport,
		Destination:         route.To,
		CallerID:            route.From,
		Variables:           req.Variables,
		Privacy:             req.Privacy,
		DTMFMode:            req.DTMFMode,
		Codecs:              req.Codecs,
		MediaEncryption:     req.MediaEncryption,
	})
	if err != nil {
		s.recordCall(ctx, "failed", route.CarrierConnectionID, route.TrunkID, route.EndpointID)
		return sqlc.Call{}, mediaError("originate call", err)
	}
	if sipCallID == "" {
		return sqlc.Call{}, apperror.NewServiceUnavailable("media server returned an empty call id", nil)
	}
	if s.admission != nil {
		if err := s.admission.Bind(ctx, route.CarrierConnectionID, leaseID, sipCallID); err != nil {
			_ = s.controller.Hangup(context.WithoutCancel(ctx), sipCallID)
			return sqlc.Call{}, apperror.NewServiceUnavailable("call quota coordination unavailable", err)
		}
		leaseID = sipCallID
	}

	call, err := s.repo.Create(ctx, organizationID, req, RouteAttribution{
		CarrierConnectionID: route.CarrierConnectionID,
		TrunkID:             route.TrunkID,
		TrunkEndpointID:     route.EndpointID,
	}, sipCallID)
	if err != nil {
		if hangupErr := s.controller.Hangup(ctx, sipCallID); hangupErr != nil {
			return sqlc.Call{}, apperror.NewInternal(
				"create call and clean up media call",
				errors.Join(err, hangupErr),
			)
		}
		return sqlc.Call{}, writeError(err, "create call")
	}

	// FreeSWITCH's synchronous originate command only returns +OK after the
	// B-leg has answered and entered the parked application. Persist that fact
	// immediately so outbound media controls are available on the returned call.
	answered, err := s.repo.MarkAnswered(ctx, organizationID, call.ID)
	if err == nil {
		s.recordCall(ctx, "answered", route.CarrierConnectionID, route.TrunkID, route.EndpointID)
		leaseID = ""
		return answered, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		latest, getErr := s.Get(ctx, organizationID, call.ID)
		if getErr == nil && isConnected(latest.State) {
			s.recordCall(ctx, "answered", route.CarrierConnectionID, route.TrunkID, route.EndpointID)
			leaseID = ""
			return latest, nil
		}
	}

	if hangupErr := s.controller.Hangup(ctx, sipCallID); hangupErr != nil {
		return sqlc.Call{}, apperror.NewInternal(
			"mark outbound call answered and clean up media call",
			errors.Join(err, hangupErr),
		)
	}
	return sqlc.Call{}, apperror.NewInternal("mark outbound call answered", err)
}

func (s *Service) Get(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return sqlc.Call{}, err
	}
	return readCall(s.repo.Get(ctx, organizationID, id))
}

func (s *Service) List(ctx context.Context, organizationID uuid.UUID, state *string, offset, limit int32) ([]sqlc.Call, error) {
	if err := validateOrganizationID(organizationID); err != nil {
		return nil, err
	}
	if offset < 0 {
		return nil, apperror.NewBadRequest("offset cannot be negative")
	}
	if limit < 1 || limit > 100 {
		return nil, apperror.NewBadRequest("limit must be between 1 and 100")
	}
	return s.repo.List(ctx, organizationID, state, offset, limit)
}

func (s *Service) Answer(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	call, externalID, err := s.controlCall(ctx, organizationID, id)
	if err != nil {
		return sqlc.Call{}, err
	}
	if err := validateControl(call, controlAnswer); err != nil {
		return sqlc.Call{}, err
	}
	if isAnswerIdempotent(call.State) {
		return call, nil
	}

	if err := s.controller.Answer(ctx, externalID); err != nil {
		return sqlc.Call{}, mediaError("answer call", err)
	}

	updated, err := s.repo.MarkAnswered(ctx, organizationID, call.ID)
	if err == nil {
		if call.CarrierConnectionID != nil {
			s.recordCall(ctx, "answered", *call.CarrierConnectionID, uuidValue(call.TrunkID), uuidValue(call.TrunkEndpointID))
		}
		return updated, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return s.Get(ctx, organizationID, call.ID)
	}
	return sqlc.Call{}, writeError(err, "answer call")
}

func (s *Service) Hangup(ctx context.Context, organizationID, id uuid.UUID) (sqlc.Call, error) {
	call, externalID, err := s.controlCall(ctx, organizationID, id)
	if err != nil {
		return sqlc.Call{}, err
	}
	if isTerminal(call.State) {
		return call, nil
	}

	if err := s.controller.Hangup(ctx, externalID); err != nil {
		return sqlc.Call{}, mediaError("hang up call", err)
	}
	if s.admission != nil && call.CarrierConnectionID != nil {
		if err := s.admission.Release(ctx, *call.CarrierConnectionID, externalID); err != nil {
			return sqlc.Call{}, apperror.NewServiceUnavailable("release call quota lease", err)
		}
	}

	var updated sqlc.Call
	if isPreAnswer(call.State) {
		updated, err = s.repo.MarkCancelled(ctx, organizationID, call.ID, nil)
	} else if isConnected(call.State) {
		updated, err = s.repo.MarkCompleted(ctx, organizationID, call.ID, nil)
	} else {
		return sqlc.Call{}, invalidControlState(controlHangup, call.State)
	}
	if err == nil {
		if call.CarrierConnectionID != nil {
			metricState := "failed"
			if updated.State == string(StateCompleted) {
				metricState = "completed"
			}
			s.recordCall(ctx, metricState, *call.CarrierConnectionID, uuidValue(call.TrunkID), uuidValue(call.TrunkEndpointID))
		}
		return updated, nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		return s.Get(ctx, organizationID, call.ID)
	}
	return sqlc.Call{}, writeError(err, "hang up call")
}

func (s *Service) Transfer(ctx context.Context, org, id uuid.UUID, req TransferRequest) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlTransfer); err != nil {
		return err
	}
	if req.Destination, err = require(req.Destination, "destination"); err != nil {
		return err
	}
	return mediaError("transfer call", s.controller.Transfer(ctx, externalID, req))
}

func (s *Service) Hold(ctx context.Context, org, id uuid.UUID) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlHold); err != nil {
		return err
	}
	if err := validateMediaState(call); err != nil {
		return err
	}
	if isHeld(call) {
		return nil
	}

	if err := s.controller.Hold(ctx, externalID); err != nil {
		return mediaError("hold call", err)
	}

	_, err = s.repo.MarkHeld(ctx, org, call.ID)
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		latest, getErr := s.Get(ctx, org, call.ID)
		if getErr == nil && isHeld(latest) {
			return nil
		}
		if getErr != nil {
			return getErr
		}
		return invalidControlState(controlHold, latest.State)
	}
	return writeError(err, "hold call")
}

func (s *Service) Unhold(ctx context.Context, org, id uuid.UUID) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlUnhold); err != nil {
		return err
	}
	if err := validateMediaState(call); err != nil {
		return err
	}
	if isMediaActive(call) {
		return nil
	}

	if err := s.controller.Unhold(ctx, externalID); err != nil {
		return mediaError("resume call", err)
	}

	_, err = s.repo.MarkResumed(ctx, org, call.ID)
	if err == nil {
		return nil
	}
	if errors.Is(err, pgx.ErrNoRows) {
		latest, getErr := s.Get(ctx, org, call.ID)
		if getErr == nil && isMediaActive(latest) {
			return nil
		}
		if getErr != nil {
			return getErr
		}
		return invalidControlState(controlUnhold, latest.State)
	}
	return writeError(err, "resume call")
}

func (s *Service) Play(ctx context.Context, org, id uuid.UUID, req PlayRequest) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlPlay); err != nil {
		return err
	}
	if req.Path, err = require(req.Path, "path"); err != nil {
		return err
	}
	return mediaError("play audio", s.controller.Play(ctx, externalID, req.Path))
}

func (s *Service) Stop(ctx context.Context, org, id uuid.UUID) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlStop); err != nil {
		return err
	}
	return mediaError("stop audio", s.controller.Stop(ctx, externalID))
}

func (s *Service) Record(ctx context.Context, org, id uuid.UUID, req RecordRequest) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlRecord); err != nil {
		return err
	}
	if err := validateRecordRequest(&req); err != nil {
		return err
	}
	return mediaError("record call", s.controller.Record(ctx, externalID, req))
}

func (s *Service) DTMF(ctx context.Context, org, id uuid.UUID, req DTMFRequest) error {
	call, externalID, err := s.controlCall(ctx, org, id)
	if err != nil {
		return err
	}
	if err := validateControl(call, controlDTMF); err != nil {
		return err
	}
	if req.Digits, err = require(req.Digits, "digits"); err != nil {
		return err
	}
	return mediaError("send DTMF", s.controller.DTMF(ctx, externalID, req.Digits))
}

func (s *Service) controlCall(ctx context.Context, org, id uuid.UUID) (sqlc.Call, string, error) {
	call, err := s.Get(ctx, org, id)
	if err != nil {
		return sqlc.Call{}, "", err
	}
	if call.SipCallID == nil || *call.SipCallID == "" {
		return sqlc.Call{}, "", apperror.NewConflict("call has no media-server id")
	}
	return call, *call.SipCallID, nil
}

func routeError(err error) error {
	if errors.Is(err, routing.ErrNoRoute) {
		return apperror.NewConflict("no outbound route available")
	}
	if errors.Is(err, routing.ErrCallerIdentity) {
		return apperror.NewForbidden("caller identity is not authorized for this trunk")
	}
	return apperror.NewInternal("resolve outbound route", err)
}

func (s *Service) recordCall(ctx context.Context, state string, carrier, trunk, endpoint uuid.UUID) {
	if s.metrics != nil {
		s.metrics.Call(ctx, state, carrier, trunk, endpoint)
	}
}

func (s *Service) recordLimitRejection(ctx context.Context, err error, carrier uuid.UUID) {
	if s.metrics == nil {
		return
	}
	reason := "coordination"
	switch {
	case errors.Is(err, ErrCPSLimit):
		reason = "cps"
	case errors.Is(err, ErrConcurrentLimit):
		reason = "concurrent"
	case errors.Is(err, ErrDailyLimit):
		reason = "daily_minutes"
	}
	s.metrics.LimitRejection(ctx, reason, carrier)
}

func uuidValue(value *uuid.UUID) uuid.UUID {
	if value == nil {
		return uuid.Nil
	}
	return *value
}

func admissionError(err error) error {
	switch {
	case errors.Is(err, ErrCPSLimit):
		return apperror.NewTooManyRequests("carrier calls-per-second limit reached")
	case errors.Is(err, ErrConcurrentLimit):
		return apperror.NewTooManyRequests("carrier concurrent-call limit reached")
	case errors.Is(err, ErrDailyLimit):
		return apperror.NewTooManyRequests("carrier daily-minute limit reached")
	default:
		return apperror.NewServiceUnavailable("call quota coordination unavailable", err)
	}
}

func readCall(call sqlc.Call, err error) (sqlc.Call, error) {
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.Call{}, apperror.NewNotFound("call not found")
	}
	if err != nil {
		return sqlc.Call{}, apperror.NewInternal("get call", err)
	}
	return call, nil
}

func writeError(err error, message string) error {
	if errors.Is(err, pgx.ErrNoRows) {
		return apperror.NewNotFound("call not found")
	}
	if err != nil {
		return apperror.NewInternal(message, err)
	}
	return nil
}

func mediaError(message string, err error) error {
	if err != nil {
		return apperror.NewServiceUnavailable(message, err)
	}
	return nil
}

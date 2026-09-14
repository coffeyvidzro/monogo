package calls

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

func (s *Service) EnsureInbound(ctx context.Context, event InboundCallEvent) error {
	if _, err := s.repo.GetBySIPCallID(ctx, event.OrganizationID, event.ChannelID); err == nil {
		return nil
	} else if !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("find inbound call: %w", err)
	}

	limits, err := s.repo.InboundCallLimits(ctx, event)
	if err != nil {
		_ = s.controller.Hangup(context.WithoutCancel(ctx), event.ChannelID)
		return fmt.Errorf("validate inbound call context: %w", err)
	}

	s.recordCall(ctx, "attempted", event.CarrierConnectionID, uuid.Nil, uuid.Nil)
	if s.admission != nil {
		leaseID, err := s.admission.Admit(ctx, limits)
		if err != nil {
			s.recordLimitRejection(ctx, err, event.CarrierConnectionID)
			return s.rejectInbound(ctx, event, "CALL_LIMIT_REJECTED", admissionError(err))
		}
		if err := s.admission.Bind(ctx, event.CarrierConnectionID, leaseID, event.ChannelID); err != nil {
			_ = s.admission.Release(context.WithoutCancel(ctx), event.CarrierConnectionID, leaseID)
			return s.rejectInbound(ctx, event, "CALL_QUOTA_UNAVAILABLE", fmt.Errorf("bind inbound call quota lease: %w", err))
		}
	}

	if _, err := s.repo.CreateInbound(ctx, event); err == nil {
		return nil
	} else if _, getErr := s.repo.GetBySIPCallID(ctx, event.OrganizationID, event.ChannelID); getErr == nil {
		// A concurrent or replayed event may have won the unique-key race.
		return nil
	} else {
		if s.admission != nil {
			_ = s.admission.Release(context.WithoutCancel(ctx), event.CarrierConnectionID, event.ChannelID)
		}
		return fmt.Errorf("create inbound call: %w", err)
	}
}

func (s *Service) rejectInbound(ctx context.Context, event InboundCallEvent, reason string, cause error) error {
	call, err := s.repo.CreateInbound(ctx, event)
	if err != nil {
		return errors.Join(cause, fmt.Errorf("persist rejected inbound call: %w", err))
	}
	if _, err := s.repo.MarkFailed(context.WithoutCancel(ctx), event.OrganizationID, call.ID, &reason); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return errors.Join(cause, fmt.Errorf("mark rejected inbound call failed: %w", err))
	}
	if err := s.controller.Hangup(context.WithoutCancel(ctx), event.ChannelID); err != nil {
		return errors.Join(cause, fmt.Errorf("hang up rejected inbound call: %w", err))
	}
	s.recordCall(ctx, "failed", event.CarrierConnectionID, uuid.Nil, uuid.Nil)
	return nil
}

func (s *Service) MarkInboundAnswered(ctx context.Context, event InboundCallEvent) error {
	if err := s.EnsureInbound(ctx, event); err != nil {
		return err
	}

	call, err := s.repo.GetBySIPCallID(ctx, event.OrganizationID, event.ChannelID)
	if err != nil {
		return fmt.Errorf("get inbound call for answer: %w", err)
	}
	if call.State == "answered" || call.State == "active" || isTerminal(call.State) {
		return nil
	}

	if _, err := s.repo.MarkAnswered(ctx, event.OrganizationID, call.ID); err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("mark inbound call answered: %w", err)
	}
	s.recordCall(ctx, "answered", event.CarrierConnectionID, uuid.Nil, uuid.Nil)
	return nil
}

func (s *Service) FinishInbound(ctx context.Context, event InboundCallEvent) error {
	if err := s.EnsureInbound(ctx, event); err != nil {
		return err
	}

	call, err := s.repo.GetBySIPCallID(ctx, event.OrganizationID, event.ChannelID)
	if err != nil {
		return fmt.Errorf("get inbound call for hangup: %w", err)
	}
	if isTerminal(call.State) {
		return nil
	}

	if event.WasAnswered && (call.State == "initiating" || call.State == "ringing") {
		if updated, answerErr := s.repo.MarkAnswered(ctx, event.OrganizationID, call.ID); answerErr == nil {
			call = updated
		} else if !errors.Is(answerErr, pgx.ErrNoRows) {
			return fmt.Errorf("reconcile inbound answer before hangup: %w", answerErr)
		}
	}

	var reason *string
	if event.HangupCause != "" {
		reason = &event.HangupCause
	}

	var finishErr error
	if s.admission != nil && call.CarrierConnectionID != nil {
		if err := s.admission.Release(ctx, *call.CarrierConnectionID, event.ChannelID); err != nil {
			return fmt.Errorf("release inbound call quota lease: %w", err)
		}
	}
	switch {
	case event.WasAnswered || call.State == "answered" || call.State == "active":
		_, finishErr = s.repo.MarkCompleted(ctx, event.OrganizationID, call.ID, reason)
	case cancelledHangupCause(event.HangupCause):
		_, finishErr = s.repo.MarkCancelled(ctx, event.OrganizationID, call.ID, reason)
	default:
		_, finishErr = s.repo.MarkFailed(ctx, event.OrganizationID, call.ID, reason)
	}
	if finishErr != nil && !errors.Is(finishErr, pgx.ErrNoRows) {
		return fmt.Errorf("finish inbound call: %w", finishErr)
	}
	state := "failed"
	if event.WasAnswered || call.State == "answered" || call.State == "active" {
		state = "completed"
	}
	s.recordCall(ctx, state, event.CarrierConnectionID, uuid.Nil, uuid.Nil)
	return nil
}

func (s *Service) MarkMediaHeld(ctx context.Context, channelID string) error {
	return s.reconcileMediaState(ctx, channelID, MediaStateHeld)
}

func (s *Service) MarkMediaResumed(ctx context.Context, channelID string) error {
	return s.reconcileMediaState(ctx, channelID, MediaStateActive)
}

func (s *Service) reconcileMediaState(ctx context.Context, channelID string, target CallMediaState) error {
	call, err := s.repo.GetBySIPCallIDGlobal(ctx, channelID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil
	}
	if err != nil {
		return fmt.Errorf("get call for media state reconciliation: %w", err)
	}
	if !isConnected(call.State) || CallMediaState(call.MediaState) == target {
		return nil
	}

	switch target {
	case MediaStateHeld:
		_, err = s.repo.MarkHeld(ctx, call.OrganizationID, call.ID)
	case MediaStateActive:
		_, err = s.repo.MarkResumed(ctx, call.OrganizationID, call.ID)
	default:
		return fmt.Errorf("unsupported media state: %s", target)
	}
	if err != nil && !errors.Is(err, pgx.ErrNoRows) {
		return fmt.Errorf("reconcile call media state: %w", err)
	}
	return nil
}

func cancelledHangupCause(cause string) bool {
	switch strings.ToUpper(strings.TrimSpace(cause)) {
	case "NORMAL_CLEARING", "ORIGINATOR_CANCEL":
		return true
	default:
		return false
	}
}

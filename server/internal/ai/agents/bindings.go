package agents

import (
	"context"
	"errors"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/coffeyvidzro/monogo/pkg/apperror"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type CreateBindingRequest struct {
	PhoneNumberID *uuid.UUID `json:"phone_number_id,omitempty"`
	SIPDomainID   *uuid.UUID `json:"sip_domain_id,omitempty"`
	SubscriberID  *uuid.UUID `json:"subscriber_id,omitempty"`
}

type BindingResponse struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	VoiceAgentID   uuid.UUID  `json:"voice_agent_id"`
	PhoneNumberID  *uuid.UUID `json:"phone_number_id,omitempty"`
	SIPDomainID    *uuid.UUID `json:"sip_domain_id,omitempty"`
	SubscriberID   *uuid.UUID `json:"subscriber_id,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
}

func (r *Repository) CreateBinding(ctx context.Context, organizationID, agentID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceAgentBinding, error) {
	return r.queries.CreateVoiceAgentBinding(ctx, sqlc.CreateVoiceAgentBindingParams{
		OrganizationID: organizationID,
		PhoneNumberID:  req.PhoneNumberID,
		SipDomainID:    req.SIPDomainID,
		SubscriberID:   req.SubscriberID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentBinding, error) {
	return r.queries.ListVoiceAgentBindingsByAgentID(ctx, sqlc.ListVoiceAgentBindingsByAgentIDParams{
		OrganizationID: organizationID,
		VoiceAgentID:   agentID,
	})
}

func (r *Repository) DeleteBinding(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	return r.queries.DeleteVoiceAgentBinding(ctx, sqlc.DeleteVoiceAgentBindingParams{
		ID: id, OrganizationID: organizationID, VoiceAgentID: agentID,
	})
}

func (s *Service) CreateBinding(ctx context.Context, organizationID, agentID uuid.UUID, req CreateBindingRequest) (sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return sqlc.VoiceAgentBinding{}, err
	}
	if err := validateBindingTarget(req); err != nil {
		return sqlc.VoiceAgentBinding{}, err
	}
	binding, err := s.repo.CreateBinding(ctx, organizationID, agentID, req)
	if errors.Is(err, pgx.ErrNoRows) {
		return sqlc.VoiceAgentBinding{}, apperror.NewNotFound("voice agent or telephony target not found")
	}
	if conflict(err) {
		return sqlc.VoiceAgentBinding{}, apperror.NewConflict("telephony target already has a voice agent")
	}
	if err != nil {
		return sqlc.VoiceAgentBinding{}, apperror.NewInternal("create voice agent binding", err)
	}
	return binding, nil
}

func (s *Service) ListBindings(ctx context.Context, organizationID, agentID uuid.UUID) ([]sqlc.VoiceAgentBinding, error) {
	if err := validateIDs(organizationID, agentID); err != nil {
		return nil, err
	}
	if _, err := s.Get(ctx, organizationID, agentID); err != nil {
		return nil, err
	}
	bindings, err := s.repo.ListBindings(ctx, organizationID, agentID)
	if err != nil {
		return nil, apperror.NewInternal("list voice agent bindings", err)
	}
	return bindings, nil
}

func (s *Service) DeleteBinding(ctx context.Context, organizationID, agentID, id uuid.UUID) error {
	if err := validateIDs(organizationID, agentID); err != nil {
		return err
	}
	if id == uuid.Nil {
		return apperror.NewBadRequest("binding id is required")
	}
	return writeError(s.repo.DeleteBinding(ctx, organizationID, agentID, id), "delete voice agent binding")
}

func validateBindingTarget(req CreateBindingRequest) error {
	count := 0
	for _, id := range []*uuid.UUID{req.PhoneNumberID, req.SIPDomainID, req.SubscriberID} {
		if id == nil {
			continue
		}
		if *id == uuid.Nil {
			return apperror.NewBadRequest("binding target id is invalid")
		}
		count++
	}
	if count != 1 {
		return apperror.NewBadRequest("exactly one telephony binding target is required")
	}
	return nil
}

func bindingResponse(binding sqlc.VoiceAgentBinding) BindingResponse {
	return BindingResponse{
		ID:             binding.ID,
		OrganizationID: binding.OrganizationID,
		VoiceAgentID:   binding.VoiceAgentID,
		PhoneNumberID:  binding.PhoneNumberID,
		SIPDomainID:    binding.SipDomainID,
		SubscriberID:   binding.SubscriberID,
		CreatedAt:      pgconv.TimestamptzToTime(binding.CreatedAt),
	}
}

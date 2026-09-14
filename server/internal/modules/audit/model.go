package audit

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/security/authn"
)

type Actor struct {
	Type string
	ID   uuid.UUID
}

type Event struct {
	ID             uuid.UUID       `json:"id"`
	OrganizationID uuid.UUID       `json:"organization_id"`
	ActorType      string          `json:"actor_type"`
	ActorID        uuid.UUID       `json:"actor_id"`
	Action         string          `json:"action"`
	TargetType     string          `json:"target_type"`
	TargetID       uuid.UUID       `json:"target_id"`
	Metadata       json.RawMessage `json:"metadata"`
	OccurredAt     time.Time       `json:"occurred_at"`
}

func ActorFromContext(ctx context.Context) (Actor, error) {
	principal, ok := authn.PrincipalFromContext(ctx)
	if !ok {
		return Actor{}, fmt.Errorf("authenticated audit actor is required")
	}
	return Actor{Type: string(principal.Subject.Type), ID: principal.Subject.ID}, nil
}

func NewEvent(organizationID uuid.UUID, actor Actor, action, targetType string, targetID uuid.UUID, metadata any) (Event, error) {
	if organizationID == uuid.Nil || actor.ID == uuid.Nil || actor.Type == "" || action == "" || targetType == "" || targetID == uuid.Nil {
		return Event{}, fmt.Errorf("complete audit event attribution is required")
	}
	encoded, err := json.Marshal(metadata)
	if err != nil {
		return Event{}, fmt.Errorf("encode audit metadata: %w", err)
	}
	return Event{OrganizationID: organizationID, ActorType: actor.Type, ActorID: actor.ID, Action: action, TargetType: targetType, TargetID: targetID, Metadata: encoded}, nil
}

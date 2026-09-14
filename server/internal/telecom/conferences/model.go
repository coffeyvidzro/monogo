package conferences

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type CreateRequest struct {
	ApplicationID *uuid.UUID `json:"application_id,omitempty"`
	Name          string     `json:"name"`
}

type AddParticipantRequest struct {
	CallParticipantID *uuid.UUID `json:"call_participant_id,omitempty"`
}

type State string

const (
	StateActive State = "active"
	StateEnded  State = "ended"
)

type ParticipantState string

const (
	ParticipantJoining ParticipantState = "joining"
	ParticipantJoined  ParticipantState = "joined"
	ParticipantLeft    ParticipantState = "left"
	ParticipantFailed  ParticipantState = "failed"
)

type EventType string

const (
	EventConferenceCreated EventType = "conference.created"
	EventConferenceEnded   EventType = "conference.ended"
	EventParticipantJoined EventType = "conference.participant.joined"
	EventParticipantLeft   EventType = "conference.participant.left"
	EventParticipantFailed EventType = "conference.participant.failed"
)

type Event struct {
	EventType      EventType  `json:"event_type"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ConferenceID   uuid.UUID  `json:"conference_id"`
	ParticipantID  *uuid.UUID `json:"participant_id,omitempty"`
	Resource       any        `json:"resource"`
	OccurredAt     time.Time  `json:"occurred_at"`
}

type ConferenceResponse struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	ApplicationID  *uuid.UUID `json:"application_id,omitempty"`
	Name           string     `json:"name"`
	State          string     `json:"state"`
	StartedAt      *time.Time `json:"started_at,omitempty"`
	EndedAt        *time.Time `json:"ended_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

type ParticipantResponse struct {
	ID                uuid.UUID  `json:"id"`
	OrganizationID    uuid.UUID  `json:"organization_id"`
	ConferenceID      uuid.UUID  `json:"conference_id"`
	CallParticipantID *uuid.UUID `json:"call_participant_id,omitempty"`
	State             string     `json:"state"`
	Muted             bool       `json:"muted"`
	Deaf              bool       `json:"deaf"`
	Speaking          bool       `json:"speaking"`
	JoinedAt          *time.Time `json:"joined_at,omitempty"`
	LeftAt            *time.Time `json:"left_at,omitempty"`
	CreatedAt         time.Time  `json:"created_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

func conferenceResponse(value sqlc.Conference) ConferenceResponse {
	return ConferenceResponse{
		ID:             value.ID,
		OrganizationID: value.OrganizationID,
		ApplicationID:  value.ApplicationID,
		Name:           value.Name,
		State:          value.State,
		StartedAt:      pgconv.TimestamptzToTimePtr(value.StartedAt),
		EndedAt:        pgconv.TimestamptzToTimePtr(value.EndedAt),
		CreatedAt:      pgconv.TimestamptzToTime(value.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(value.UpdatedAt),
	}
}

func participantResponse(value sqlc.ConferenceParticipant) ParticipantResponse {
	return ParticipantResponse{
		ID:                value.ID,
		OrganizationID:    value.OrganizationID,
		ConferenceID:      value.ConferenceID,
		CallParticipantID: value.CallParticipantID,
		State:             value.State,
		Muted:             value.Muted,
		Deaf:              value.Deaf,
		Speaking:          value.Speaking,
		JoinedAt:          pgconv.TimestamptzToTimePtr(value.JoinedAt),
		LeftAt:            pgconv.TimestamptzToTimePtr(value.LeftAt),
		CreatedAt:         pgconv.TimestamptzToTime(value.CreatedAt),
		UpdatedAt:         pgconv.TimestamptzToTime(value.UpdatedAt),
	}
}

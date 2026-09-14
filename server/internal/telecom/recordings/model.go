package recordings

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Status string

const (
	StatusRecording Status = "recording"
	StatusCompleted Status = "completed"
	StatusFailed    Status = "failed"
	StatusDeleted   Status = "deleted"
)

type EventType string

const (
	EventRecordingStarted   EventType = "recording.started"
	EventRecordingCompleted EventType = "recording.completed"
	EventRecordingFailed    EventType = "recording.failed"
	EventRecordingDeleted   EventType = "recording.deleted"
)

type LifecycleEvent struct {
	ChannelID  string
	Path       string
	OccurredAt time.Time
}

type Event struct {
	EventType      EventType         `json:"event_type"`
	OrganizationID uuid.UUID         `json:"organization_id"`
	RecordingID    uuid.UUID         `json:"recording_id"`
	CallID         uuid.UUID         `json:"call_id"`
	Resource       RecordingResponse `json:"resource"`
	OccurredAt     time.Time         `json:"occurred_at"`
}

type RecordingResponse struct {
	ID              uuid.UUID  `json:"id"`
	OrganizationID  uuid.UUID  `json:"organization_id"`
	CallID          uuid.UUID  `json:"call_id"`
	Status          string     `json:"status"`
	FileSizeBytes   *int64     `json:"file_size_bytes,omitempty"`
	Format          *string    `json:"format,omitempty"`
	DurationSeconds *int32     `json:"duration_seconds,omitempty"`
	StartedAt       *time.Time `json:"started_at,omitempty"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type PlaybackResponse struct {
	URL       string    `json:"url"`
	ExpiresAt time.Time `json:"expires_at"`
}

func recordingResponse(recording sqlc.Recording) RecordingResponse {
	return RecordingResponse{
		ID:              recording.ID,
		OrganizationID:  recording.OrganizationID,
		CallID:          recording.CallID,
		Status:          recording.Status,
		FileSizeBytes:   recording.FileSizeBytes,
		Format:          recording.Format,
		DurationSeconds: recording.DurationSeconds,
		StartedAt:       pgconv.TimestamptzToTimePtr(recording.StartedAt),
		CompletedAt:     pgconv.TimestamptzToTimePtr(recording.CompletedAt),
		CreatedAt:       pgconv.TimestamptzToTime(recording.CreatedAt),
		UpdatedAt:       pgconv.TimestamptzToTime(recording.UpdatedAt),
	}
}

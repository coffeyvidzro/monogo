package runtimes

import (
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusDisabled = "disabled"

	onlineWindow = 30 * time.Second
)

type Runtime struct {
	ID             uuid.UUID
	OrganizationID uuid.UUID
	Name           string
	Status         string
	Version        *string
	Region         *string
	Capabilities   []string
	Capacity       int32
	ActiveSessions int32
	LastSeenAt     *time.Time
	CreatedAt      time.Time
	UpdatedAt      time.Time
}

type CreateRequest struct {
	Name   string  `json:"name"`
	Region *string `json:"region,omitempty"`
}

type HeartbeatRequest struct {
	Version        string   `json:"version"`
	Capabilities   []string `json:"capabilities"`
	Capacity       int32    `json:"capacity"`
	ActiveSessions int32    `json:"active_sessions"`
}

type Response struct {
	ID             uuid.UUID  `json:"id"`
	OrganizationID uuid.UUID  `json:"organization_id"`
	Name           string     `json:"name"`
	Status         string     `json:"status"`
	Version        *string    `json:"version,omitempty"`
	Region         *string    `json:"region,omitempty"`
	Capabilities   []string   `json:"capabilities"`
	Capacity       int32      `json:"capacity"`
	ActiveSessions int32      `json:"active_sessions"`
	Online         bool       `json:"online"`
	LastSeenAt     *time.Time `json:"last_seen_at,omitempty"`
	CreatedAt      time.Time  `json:"created_at"`
	UpdatedAt      time.Time  `json:"updated_at"`
}

func response(value Runtime, now time.Time) Response {
	return Response{
		ID:             value.ID,
		OrganizationID: value.OrganizationID,
		Name:           value.Name,
		Status:         value.Status,
		Version:        value.Version,
		Region:         value.Region,
		Capabilities:   value.Capabilities,
		Capacity:       value.Capacity,
		ActiveSessions: value.ActiveSessions,
		Online:         value.Status == StatusActive && value.LastSeenAt != nil && now.Sub(*value.LastSeenAt) <= onlineWindow,
		LastSeenAt:     value.LastSeenAt,
		CreatedAt:      value.CreatedAt,
		UpdatedAt:      value.UpdatedAt,
	}
}

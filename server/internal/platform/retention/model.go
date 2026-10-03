package retention

import (
	"time"

	"github.com/google/uuid"
)

const (
	ResourceRecordings    = "recordings"
	ResourceConversations = "conversations"
	ResourceAuditEvents   = "audit_events"
)

type Policy struct {
	OrganizationID uuid.UUID `json:"organization_id"`
	Resource       string    `json:"resource"`
	RetentionDays  int32     `json:"retention_days"`
	Enabled        bool      `json:"enabled"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

type UpsertRequest struct {
	RetentionDays int32 `json:"retention_days"`
	Enabled       bool  `json:"enabled"`
}

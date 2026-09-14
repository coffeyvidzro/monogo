package session

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type Response struct {
	ID         uuid.UUID `json:"id"`
	UserID     uuid.UUID `json:"user_id"`
	IPAddress  *string   `json:"ip_address,omitempty"`
	UserAgent  *string   `json:"user_agent,omitempty"`
	ExpiresAt  string    `json:"expires_at"`
	LastSeenAt string    `json:"last_seen_at,omitempty"`
	CreatedAt  string    `json:"created_at"`
}

func toResponse(session sqlc.Session) Response {
	response := Response{
		ID:        session.ID,
		UserID:    session.UserID,
		UserAgent: session.UserAgent,
	}

	if session.IpAddress != nil {
		ip := session.IpAddress.String()
		response.IPAddress = &ip
	}

	if session.ExpiresAt.Valid {
		response.ExpiresAt = pgconv.
			TimestamptzToTime(session.ExpiresAt).
			Format(time.RFC3339)
	}

	if session.LastSeenAt.Valid {
		response.LastSeenAt = pgconv.
			TimestamptzToTime(session.LastSeenAt).
			Format(time.RFC3339)
	}

	if session.CreatedAt.Valid {
		response.CreatedAt = pgconv.
			TimestamptzToTime(session.CreatedAt).
			Format(time.RFC3339)
	}

	return response
}

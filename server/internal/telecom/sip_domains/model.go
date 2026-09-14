package sip_domains

import (
	"time"

	"github.com/google/uuid"
	"github.com/leamout/leamout/internal/database/pgconv"
	"github.com/leamout/leamout/internal/database/sqlc"
)

type CreateRequest struct {
	Domain string `json:"domain"`
}

type UpdateRequest struct {
	Domain *string `json:"domain,omitempty"`
}

type Response struct {
	ID             uuid.UUID `json:"id"`
	OrganizationID uuid.UUID `json:"organization_id"`
	Domain         string    `json:"domain"`
	Status         string    `json:"status"`
	CreatedAt      time.Time `json:"created_at"`
	UpdatedAt      time.Time `json:"updated_at"`
}

func response(domain sqlc.SipDomain) Response {
	return Response{
		ID:             domain.ID,
		OrganizationID: domain.OrganizationID,
		Domain:         domain.Domain,
		Status:         domain.Status,
		CreatedAt:      pgconv.TimestamptzToTime(domain.CreatedAt),
		UpdatedAt:      pgconv.TimestamptzToTime(domain.UpdatedAt),
	}
}

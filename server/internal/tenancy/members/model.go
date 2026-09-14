package members

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	UserID    uuid.UUID `json:"user_id"`
	Role      string    `json:"role"`
	Status    string    `json:"status"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	UserID uuid.UUID `json:"user_id"`
	Role   string    `json:"role"`
}

type UpdateRequest struct {
	Role string `json:"role"`
}

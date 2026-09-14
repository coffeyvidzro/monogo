package organization

import (
	"time"

	"github.com/google/uuid"
)

type Response struct {
	ID        uuid.UUID `json:"id"`
	Name      string    `json:"name"`
	Status    string    `json:"status"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Name string `json:"name"`
}

type UpdateRequest struct {
	Name *string `json:"name,omitempty"`
}

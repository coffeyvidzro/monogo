package plans

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	StatusActive   = "active"
	StatusArchived = "archived"
)

var (
	ErrInvalidInput = errors.New("invalid plan input")
	ErrNotFound     = errors.New("plan not found")
	ErrConflict     = errors.New("plan conflict")
)

type Plan struct {
	ID           uuid.UUID `json:"id"`
	Code         string    `json:"code"`
	Name         string    `json:"name"`
	Currency     string    `json:"currency"`
	Interval     string    `json:"interval"`
	AmountMicros int64     `json:"amount_micros"`
	Status       string    `json:"status"`
	CreatedAt    time.Time `json:"created_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}

type CreateRequest struct {
	Code         string
	Name         string
	Currency     string
	AmountMicros int64
}

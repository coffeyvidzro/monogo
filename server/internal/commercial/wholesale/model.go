package wholesale

import (
	"errors"
	"time"

	"github.com/google/uuid"
)

const (
	DirectionInbound  = "inbound"
	DirectionOutbound = "outbound"
)

var (
	ErrProviderCDRNotFound = errors.New("provider CDR not found")
	ErrProviderCDRRejected = errors.New("provider CDR was not accepted")
	ErrProviderCDRConflict = errors.New("provider CDR conflicts with existing record")
	ErrChargeNotFound      = errors.New("wholesale charge not found")
	ErrChargeConflict      = errors.New("wholesale charge conflicts with existing record")
)

type ProviderCDR struct {
	ID                  uuid.UUID
	ProviderID          uuid.UUID
	CarrierConnectionID uuid.UUID
	CallID              uuid.UUID
	ProviderCDRID       string
	Direction           string
	Source              *string
	Destination         *string
	StartedAt           time.Time
	AnsweredAt          *time.Time
	EndedAt             time.Time
	DurationSeconds     int64
	BillableSeconds     int64
	RawPayload          []byte
	ReceivedAt          time.Time
	CreatedAt           time.Time
}

type Charge struct {
	ID              uuid.UUID
	ProviderCDRID   uuid.UUID
	Currency        string
	RateMicros      int64
	BillableSeconds int64
	AmountMicros    int64
	RatedAt         time.Time
	CreatedAt       time.Time
}

type RecordCDRRequest struct {
	ProviderID          uuid.UUID
	CarrierConnectionID uuid.UUID
	CallID              uuid.UUID
	ProviderCDRID       string
	Direction           string
	Source              *string
	Destination         *string
	StartedAt           time.Time
	AnsweredAt          *time.Time
	EndedAt             time.Time
	DurationSeconds     int64
	BillableSeconds     int64
	RawPayload          []byte
	ReceivedAt          time.Time
}

type RecordChargeRequest struct {
	ProviderCDRID   uuid.UUID
	Currency        string
	RateMicros      int64
	BillableSeconds int64
	AmountMicros    int64
	RatedAt         time.Time
}

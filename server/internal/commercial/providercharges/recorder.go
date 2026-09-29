package providercharges

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

var ErrConflict = errors.New("provider charge conflicts with existing record")

var products = map[string]struct{}{
	"sms_outbound":      {},
	"whatsapp_outbound": {},
	"number_purchase":   {},
	"number_renewal":    {},
}

type RecordProviderChargeRequest struct {
	ProviderID         uuid.UUID
	OperationID        *uuid.UUID
	ProviderRecordType string
	ProviderRecordID   string
	Product            string
	Currency           string
	AmountMicros       int64
	IncurredAt         time.Time
	RawPayload         []byte
}

// Record persists immutable supplier evidence and is idempotent by the
// provider's record identifier. Reusing that identifier with different cost
// data is rejected rather than silently corrupting margin reporting.
func (s *Service) RecordProviderCharge(ctx context.Context, req RecordProviderChargeRequest) (sqlc.ProviderCharge, error) {
	if err := normalize(&req); err != nil {
		return sqlc.ProviderCharge{}, err
	}
	row, err := s.repo.queries.CreateProviderCharge(
		ctx,
		sqlc.CreateProviderChargeParams{
			ProviderRecordType: req.ProviderRecordType,
			ProviderRecordID:   req.ProviderRecordID,
			Product:            req.Product,
			Currency:           req.Currency,
			AmountMicros:       req.AmountMicros,
			IncurredAt:         pgconv.TimeToTimestamptz(req.IncurredAt),
			RawPayload:         req.RawPayload,
			OperationID:        req.OperationID,
			ProviderID:         req.ProviderID,
		},
	)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.ProviderCharge{}, fmt.Errorf("record provider charge: %w", err)
	}
	existing, lookupErr := s.repo.queries.GetProviderChargeByRecord(
		ctx,
		sqlc.GetProviderChargeByRecordParams{
			ProviderID:         req.ProviderID,
			ProviderRecordType: req.ProviderRecordType,
			ProviderRecordID:   req.ProviderRecordID,
		},
	)
	if lookupErr != nil {
		return sqlc.ProviderCharge{}, fmt.Errorf("provider charge was not accepted: %w", lookupErr)
	}
	if !equalOperationID(existing.OperationID, req.OperationID) ||
		existing.ProviderRecordType != req.ProviderRecordType ||
		existing.Product != req.Product ||
		existing.Currency != req.Currency ||
		existing.AmountMicros != req.AmountMicros ||
		!existing.IncurredAt.Time.Equal(req.IncurredAt) ||
		string(existing.RawPayload) != string(req.RawPayload) {
		return sqlc.ProviderCharge{}, ErrConflict
	}
	return existing, nil
}

func normalize(req *RecordProviderChargeRequest) error {
	if req.ProviderID == uuid.Nil {
		return fmt.Errorf("provider id is required")
	}
	req.ProviderRecordType = strings.TrimSpace(req.ProviderRecordType)
	if req.ProviderRecordType != "order" && req.ProviderRecordType != "message" &&
		req.ProviderRecordType != "invoice_item" {
		return fmt.Errorf("unsupported provider record type")
	}
	req.ProviderRecordID = strings.TrimSpace(req.ProviderRecordID)
	if req.ProviderRecordID == "" || len(req.ProviderRecordID) > 255 {
		return fmt.Errorf("provider record id must contain 1-255 characters")
	}
	if _, ok := products[req.Product]; !ok {
		return fmt.Errorf("unsupported provider charge product")
	}
	req.Currency = strings.ToUpper(strings.TrimSpace(req.Currency))
	if len(req.Currency) != 3 {
		return fmt.Errorf("currency must contain three letters")
	}
	for _, char := range req.Currency {
		if char < 'A' || char > 'Z' {
			return fmt.Errorf("currency must contain three letters")
		}
	}
	if req.AmountMicros < 0 {
		return fmt.Errorf("provider charge amount cannot be negative")
	}
	if req.IncurredAt.IsZero() {
		return fmt.Errorf("provider charge incurred time is required")
	}
	if len(req.RawPayload) == 0 {
		req.RawPayload = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(req.RawPayload, &payload); err != nil || payload == nil {
		return fmt.Errorf("provider charge payload must be a JSON object")
	}
	canonicalPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("canonicalize provider charge payload: %w", err)
	}
	req.RawPayload = canonicalPayload
	req.IncurredAt = req.IncurredAt.UTC()
	return nil
}

func equalOperationID(left *uuid.UUID, right *uuid.UUID) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}
	return *left == *right
}

package providercosts

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

var ErrConflict = errors.New("provider cost conflicts with existing record")

var products = map[string]struct{}{
	"voice":             {},
	"sms_outbound":      {},
	"whatsapp_outbound": {},
	"number_purchase":   {},
	"number_renewal":    {},
}

type RecordRequest struct {
	ProviderID       uuid.UUID
	OperationID      uuid.UUID
	ProviderRecordID string
	Product          string
	Currency         string
	AmountMicros     int64
	IncurredAt       time.Time
	RawPayload       []byte
}

type Service struct {
	queries *sqlc.Queries
	now     func() time.Time
}

func NewService(queries *sqlc.Queries) *Service {
	return &Service{
		queries: queries,
		now:     time.Now,
	}
}

// Record persists immutable supplier evidence and is idempotent by the
// provider's record identifier. Reusing that identifier with different cost
// data is rejected rather than silently corrupting margin reporting.
func (s *Service) Record(ctx context.Context, req RecordRequest) (sqlc.ProviderCost, error) {
	if err := normalize(&req); err != nil {
		return sqlc.ProviderCost{}, err
	}
	row, err := s.queries.CreateProviderCost(
		ctx,
		sqlc.CreateProviderCostParams{
			ProviderRecordID: req.ProviderRecordID,
			Product:          req.Product,
			Currency:         req.Currency,
			AmountMicros:     req.AmountMicros,
			IncurredAt:       pgconv.TimeToTimestamptz(req.IncurredAt),
			RawPayload:       req.RawPayload,
			OperationID:      req.OperationID,
			ProviderID:       req.ProviderID,
		},
	)
	if err == nil {
		return row, nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return sqlc.ProviderCost{}, fmt.Errorf("record provider cost: %w", err)
	}
	existing, lookupErr := s.queries.GetProviderCostByProviderRecord(
		ctx,
		sqlc.GetProviderCostByProviderRecordParams{
			ProviderID:       req.ProviderID,
			ProviderRecordID: req.ProviderRecordID,
		},
	)
	if lookupErr != nil {
		return sqlc.ProviderCost{}, fmt.Errorf("provider cost has no captured retail operation: %w", lookupErr)
	}
	if existing.OperationID != req.OperationID ||
		existing.Product != req.Product ||
		existing.Currency != req.Currency ||
		existing.AmountMicros != req.AmountMicros ||
		!existing.IncurredAt.Time.Equal(req.IncurredAt) ||
		string(existing.RawPayload) != string(req.RawPayload) {
		return sqlc.ProviderCost{}, ErrConflict
	}
	return existing, nil
}

func normalize(req *RecordRequest) error {
	if req.ProviderID == uuid.Nil || req.OperationID == uuid.Nil {
		return fmt.Errorf("provider id and retail operation id are required")
	}
	req.ProviderRecordID = strings.TrimSpace(req.ProviderRecordID)
	if req.ProviderRecordID == "" || len(req.ProviderRecordID) > 255 {
		return fmt.Errorf("provider record id must contain 1-255 characters")
	}
	if _, ok := products[req.Product]; !ok {
		return fmt.Errorf("unsupported provider cost product")
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
		return fmt.Errorf("provider cost amount cannot be negative")
	}
	if req.IncurredAt.IsZero() {
		return fmt.Errorf("provider cost incurred time is required")
	}
	if len(req.RawPayload) == 0 {
		req.RawPayload = []byte("{}")
	}
	var payload map[string]any
	if err := json.Unmarshal(req.RawPayload, &payload); err != nil || payload == nil {
		return fmt.Errorf("provider cost payload must be a JSON object")
	}
	canonicalPayload, err := json.Marshal(payload)
	if err != nil {
		return fmt.Errorf("canonicalize provider cost payload: %w", err)
	}
	req.RawPayload = canonicalPayload
	req.IncurredAt = req.IncurredAt.UTC()
	return nil
}

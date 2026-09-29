package providercharges

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/coffeyvidzro/monogo/internal/database/pgconv"
	"github.com/coffeyvidzro/monogo/internal/database/sqlc"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgtype"
)

type Service struct {
	repo *Repository
	now  func() time.Time
}

func NewService(repo *Repository) *Service {
	return &Service{
		repo: repo,
		now:  time.Now,
	}
}

func (s *Service) RecordProviderCDR(
	ctx context.Context,
	req RecordCDRRequest,
) (ProviderCDR, error) {
	if err := normalizeRecordCDRRequest(&req); err != nil {
		return ProviderCDR{}, err
	}
	if req.ReceivedAt.IsZero() {
		req.ReceivedAt = s.now().UTC()
	}

	row, err := s.repo.CreateProviderCDR(
		ctx,
		sqlc.CreateProviderCDRParams{
			ProviderCdrID:       req.ProviderCDRID,
			Direction:           req.Direction,
			Source:              req.Source,
			Destination:         req.Destination,
			StartedAt:           pgconv.TimeToTimestamptz(req.StartedAt),
			AnsweredAt:          pgconv.NullableTimestamptz(req.AnsweredAt),
			EndedAt:             pgconv.TimeToTimestamptz(req.EndedAt),
			DurationSeconds:     req.DurationSeconds,
			BillableSeconds:     req.BillableSeconds,
			RawPayload:          req.RawPayload,
			ReceivedAt:          pgconv.TimeToTimestamptz(req.ReceivedAt),
			CarrierConnectionID: req.CarrierConnectionID,
			ProviderID:          req.ProviderID,
			CallID:              req.CallID,
		},
	)
	if err == nil {
		return providerCDRFromRow(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProviderCDR{}, fmt.Errorf("record provider CDR: %w", err)
	}

	existing, lookupErr := s.repo.GetProviderCDRByProviderRecord(
		ctx,
		req.ProviderID,
		req.ProviderCDRID,
	)
	if lookupErr == nil {
		if !sameProviderCDR(existing, req) {
			return ProviderCDR{}, ErrProviderCDRConflict
		}

		return providerCDRFromRow(existing), nil
	}
	if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return ProviderCDR{}, fmt.Errorf("read provider CDR replay: %w", lookupErr)
	}

	return ProviderCDR{}, ErrProviderCDRRejected
}

func (s *Service) GetProviderCDR(
	ctx context.Context,
	id uuid.UUID,
) (ProviderCDR, error) {
	if id == uuid.Nil {
		return ProviderCDR{}, ErrProviderCDRNotFound
	}

	row, err := s.repo.GetProviderCDR(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderCDR{}, ErrProviderCDRNotFound
	}
	if err != nil {
		return ProviderCDR{}, fmt.Errorf("get provider CDR: %w", err)
	}

	return providerCDRFromRow(row), nil
}

func (s *Service) ListProviderCDRsByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]ProviderCDR, error) {
	if callID == uuid.Nil {
		return nil, fmt.Errorf("call id is required")
	}

	rows, err := s.repo.ListProviderCDRsByCall(
		ctx,
		callID,
	)
	if err != nil {
		return nil, fmt.Errorf("list provider CDRs: %w", err)
	}

	result := make([]ProviderCDR, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			providerCDRFromRow(row),
		)
	}

	return result, nil
}

func (s *Service) RecordVoiceCharge(
	ctx context.Context,
	req RecordVoiceChargeRequest,
) (ProviderCharge, error) {
	if err := normalizeRecordVoiceChargeRequest(&req); err != nil {
		return ProviderCharge{}, err
	}
	if req.IncurredAt.IsZero() {
		req.IncurredAt = s.now().UTC()
	}

	row, err := s.repo.CreateCharge(
		ctx,
		sqlc.CreateVoiceProviderChargeParams{
			OperationID:     req.OperationID,
			Currency:        req.Currency,
			RateMicros:      &req.RateMicros,
			BillableSeconds: &req.BillableSeconds,
			AmountMicros:    req.AmountMicros,
			IncurredAt:      pgconv.TimeToTimestamptz(req.IncurredAt),
			ProviderCdrID:   req.ProviderCDRID,
		},
	)
	if err == nil {
		return providerChargeFromRow(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return ProviderCharge{}, fmt.Errorf("record provider charge: %w", err)
	}

	existing, lookupErr := s.repo.GetChargeByProviderCDR(
		ctx,
		req.ProviderCDRID,
	)
	if lookupErr == nil {
		if !sameVoiceCharge(existing, req) {
			return ProviderCharge{}, ErrProviderChargeConflict
		}

		return providerChargeFromRow(existing), nil
	}
	if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return ProviderCharge{}, fmt.Errorf("read provider charge replay: %w", lookupErr)
	}

	return ProviderCharge{}, ErrProviderChargeNotFound
}

func (s *Service) GetProviderCharge(
	ctx context.Context,
	id uuid.UUID,
) (ProviderCharge, error) {
	if id == uuid.Nil {
		return ProviderCharge{}, ErrProviderChargeNotFound
	}

	row, err := s.repo.GetProviderCharge(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return ProviderCharge{}, ErrProviderChargeNotFound
	}
	if err != nil {
		return ProviderCharge{}, fmt.Errorf("get provider charge: %w", err)
	}

	return providerChargeFromRow(row), nil
}

func (s *Service) ListProviderChargesByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]ProviderCharge, error) {
	if callID == uuid.Nil {
		return nil, fmt.Errorf("call id is required")
	}

	rows, err := s.repo.ListProviderChargesByCall(
		ctx,
		callID,
	)
	if err != nil {
		return nil, fmt.Errorf("list provider charges: %w", err)
	}

	result := make([]ProviderCharge, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			providerChargeFromRow(row),
		)
	}

	return result, nil
}

func sameProviderCDR(
	existing sqlc.ProviderCdr,
	req RecordCDRRequest,
) bool {
	return existing.ProviderID == req.ProviderID &&
		existing.CarrierConnectionID == req.CarrierConnectionID &&
		existing.CallID == req.CallID &&
		existing.ProviderCdrID == req.ProviderCDRID &&
		existing.Direction == req.Direction &&
		equalOptionalString(existing.Source, req.Source) &&
		equalOptionalString(existing.Destination, req.Destination) &&
		pgconv.TimestamptzToTime(existing.StartedAt).Equal(req.StartedAt) &&
		equalOptionalTime(existing.AnsweredAt, req.AnsweredAt) &&
		pgconv.TimestamptzToTime(existing.EndedAt).Equal(req.EndedAt) &&
		existing.DurationSeconds == req.DurationSeconds &&
		existing.BillableSeconds == req.BillableSeconds &&
		bytes.Equal(existing.RawPayload, req.RawPayload)
}

func sameVoiceCharge(
	existing sqlc.ProviderCharge,
	req RecordVoiceChargeRequest,
) bool {
	return existing.ProviderCdrID != nil && *existing.ProviderCdrID == req.ProviderCDRID &&
		existing.Currency == req.Currency &&
		existing.RateMicros != nil && *existing.RateMicros == req.RateMicros &&
		existing.BillableSeconds != nil && *existing.BillableSeconds == req.BillableSeconds &&
		existing.AmountMicros == req.AmountMicros
}

func equalOptionalString(
	left *string,
	right *string,
) bool {
	if left == nil || right == nil {
		return left == nil && right == nil
	}

	return *left == *right
}

func equalOptionalTime(
	left pgtype.Timestamptz,
	right *time.Time,
) bool {
	if !left.Valid || right == nil {
		return !left.Valid && right == nil
	}

	return left.Time.Equal(*right)
}

func providerCDRFromRow(row sqlc.ProviderCdr) ProviderCDR {
	return ProviderCDR{
		ID:                  row.ID,
		ProviderID:          row.ProviderID,
		CarrierConnectionID: row.CarrierConnectionID,
		CallID:              row.CallID,
		ProviderCDRID:       row.ProviderCdrID,
		Direction:           row.Direction,
		Source:              row.Source,
		Destination:         row.Destination,
		StartedAt:           pgconv.TimestamptzToTime(row.StartedAt),
		AnsweredAt:          pgconv.TimestamptzToTimePtr(row.AnsweredAt),
		EndedAt:             pgconv.TimestamptzToTime(row.EndedAt),
		DurationSeconds:     row.DurationSeconds,
		BillableSeconds:     row.BillableSeconds,
		RawPayload:          row.RawPayload,
		ReceivedAt:          pgconv.TimestamptzToTime(row.ReceivedAt),
		CreatedAt:           pgconv.TimestamptzToTime(row.CreatedAt),
	}
}

func providerChargeFromRow(row sqlc.ProviderCharge) ProviderCharge {
	return ProviderCharge{
		ID:              row.ID,
		ProviderID:      row.ProviderID,
		ProviderCDRID:   row.ProviderCdrID,
		OperationID:     row.OperationID,
		Currency:        row.Currency,
		RateMicros:      row.RateMicros,
		BillableSeconds: row.BillableSeconds,
		AmountMicros:    row.AmountMicros,
		IncurredAt:      pgconv.TimestamptzToTime(row.IncurredAt),
		RecordedAt:      pgconv.TimestamptzToTime(row.RecordedAt),
	}
}

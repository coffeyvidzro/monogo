package wholesale

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

func (s *Service) RecordCharge(
	ctx context.Context,
	req RecordChargeRequest,
) (Charge, error) {
	if err := normalizeRecordChargeRequest(&req); err != nil {
		return Charge{}, err
	}
	if req.RatedAt.IsZero() {
		req.RatedAt = s.now().UTC()
	}

	row, err := s.repo.CreateCharge(
		ctx,
		sqlc.CreateWholesaleChargeParams{
			Currency:        req.Currency,
			RateMicros:      req.RateMicros,
			BillableSeconds: req.BillableSeconds,
			AmountMicros:    req.AmountMicros,
			RatedAt:         pgconv.TimeToTimestamptz(req.RatedAt),
			ProviderCdrID:   req.ProviderCDRID,
		},
	)
	if err == nil {
		return chargeFromRow(row), nil
	}
	if !errors.Is(err, pgx.ErrNoRows) {
		return Charge{}, fmt.Errorf("record wholesale charge: %w", err)
	}

	existing, lookupErr := s.repo.GetChargeByProviderCDR(
		ctx,
		req.ProviderCDRID,
	)
	if lookupErr == nil {
		if !sameCharge(existing, req) {
			return Charge{}, ErrChargeConflict
		}

		return chargeFromRow(existing), nil
	}
	if !errors.Is(lookupErr, pgx.ErrNoRows) {
		return Charge{}, fmt.Errorf("read wholesale charge replay: %w", lookupErr)
	}

	return Charge{}, ErrChargeNotFound
}

func (s *Service) GetCharge(
	ctx context.Context,
	id uuid.UUID,
) (Charge, error) {
	if id == uuid.Nil {
		return Charge{}, ErrChargeNotFound
	}

	row, err := s.repo.GetCharge(
		ctx,
		id,
	)
	if errors.Is(err, pgx.ErrNoRows) {
		return Charge{}, ErrChargeNotFound
	}
	if err != nil {
		return Charge{}, fmt.Errorf("get wholesale charge: %w", err)
	}

	return chargeFromRow(row), nil
}

func (s *Service) ListChargesByCall(
	ctx context.Context,
	callID uuid.UUID,
) ([]Charge, error) {
	if callID == uuid.Nil {
		return nil, fmt.Errorf("call id is required")
	}

	rows, err := s.repo.ListChargesByCall(
		ctx,
		callID,
	)
	if err != nil {
		return nil, fmt.Errorf("list wholesale charges: %w", err)
	}

	result := make([]Charge, 0, len(rows))
	for _, row := range rows {
		result = append(
			result,
			chargeFromRow(row),
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

func sameCharge(
	existing sqlc.WholesaleCharge,
	req RecordChargeRequest,
) bool {
	return existing.ProviderCdrID == req.ProviderCDRID &&
		existing.Currency == req.Currency &&
		existing.RateMicros == req.RateMicros &&
		existing.BillableSeconds == req.BillableSeconds &&
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

func chargeFromRow(row sqlc.WholesaleCharge) Charge {
	return Charge{
		ID:              row.ID,
		ProviderCDRID:   row.ProviderCdrID,
		Currency:        row.Currency,
		RateMicros:      row.RateMicros,
		BillableSeconds: row.BillableSeconds,
		AmountMicros:    row.AmountMicros,
		RatedAt:         pgconv.TimestamptzToTime(row.RatedAt),
		CreatedAt:       pgconv.TimestamptzToTime(row.CreatedAt),
	}
}
